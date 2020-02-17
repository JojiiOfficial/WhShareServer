package main

import (
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

//Subscription the subscription a user made
type Subscription struct {
	PkID           uint32 `db:"pk_id" orm:"pk,ai"`
	SubscriptionID string `db:"subscriptionID"`
	UserID         uint32 `db:"subscriber"`
	Source         uint32 `db:"source"`
	CallbackURL    string `db:"callbackURL"`
	Time           string `db:"time"`
	IsValid        bool   `db:"isValid"`
	LastTrigger    string `db:"lastTrigger"`
}

//Notify all subscriber for a given webhook
func notifyAllSubscriber(db *dbhelper.DBhelper, webhook *Webhook, source *Source) {
	subscriptions, err := getSubscriptionsForSource(db, source.PkID)
	if err != nil {
		log.Println(err.Error())
		return
	}

	if len(subscriptions) > 0 {
		go (func() {
			startPool(db, webhook, source, subscriptions)
		})()
	} else {
		log.Println("No subscriber found!")
	}
}

//Start notifier pool
func startPool(db *dbhelper.DBhelper, webhook *Webhook, source *Source, subscriptions []Subscription) {
	pos := 0

	c := make(chan int, 1)
	c <- config.Server.WorkerCount

	for pos < len(subscriptions) {
		read := <-c
		for i := 0; i < read && pos < len(subscriptions); i++ {

			go (func(c *chan int, subscription *Subscription, webhook *Webhook, source *Source) {
				rand.Seed(time.Now().UnixNano())
				<-time.After(time.Duration(rand.Intn(999)+1) * time.Millisecond)

				subscription.Notify(webhook, source)
				*c <- 1
			})(&c, &subscriptions[pos], webhook, source)

			pos++
		}
	}
}

//Notify subscriber
func (subscription *Subscription) Notify(webhook *Webhook, source *Source) {
	client := &http.Client{
		Timeout: 20 * time.Second,
	}
	req, _ := http.NewRequest("POST", subscription.CallbackURL, strings.NewReader(webhook.Payload))

	//Load headers from webhook.Headers
	setHeadersFromStr(webhook.Headers, &req.Header)

	//Add header for client
	req.Header.Set(HeaderReceived, webhook.Received)
	req.Header.Set(HeaderSource, source.SourceID)
	req.Header.Set(HeaderSubsID, subscription.SubscriptionID)

	//Do the request
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err.Error())
	}

	if err != nil || resp.StatusCode > 299 || resp.StatusCode < 200 {
		retryService.add(subscription.PkID, source.PkID, webhook.PkID)
	} else if resp.StatusCode == http.StatusTeapot {
		//Unsubscribe
		err := removeSubscription(db, subscription.SubscriptionID)
		if err != nil {
			log.Println(err.Error())
		}
	} else {
		//Successful notification
		retryService.remove(subscription.PkID)
		log.Println("Removing subscription from retryQueue. Reason: successful notification")

		if !subscription.IsValid {
			subscription.triggerAndValidate(db)
		} else {
			subscription.trigger(db)
		}
	}
}
