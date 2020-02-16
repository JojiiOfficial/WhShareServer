package main

import (
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

//Notify all subscriber for a given webhook
func notifyAllSubscriber(db *dbhelper.DBhelper, webhook *Webhook, source *Source) {
	subscriptions, err := getValidSubscriptions(db, source.PkID)
	if err != nil {
		log.Println(err.Error())
		return
	}

	go (func() {
		const numWorkers = 4
		startPool(db, numWorkers, webhook, source, subscriptions)
	})()
}

//Start notifier pool
func startPool(db *dbhelper.DBhelper, numWorkers int, webhook *Webhook, source *Source, subscriptions []Subscription) {
	pos := 0

	c := make(chan int, 1)
	c <- numWorkers

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
		Timeout: 30 * time.Second,
	}
	req, _ := http.NewRequest("POST", subscription.CallbackURL, strings.NewReader(webhook.Payload))

	headersrn := strings.Split(webhook.Headers, "\r\n")
	for _, v := range headersrn {
		if !strings.Contains(v, "=") {
			continue
		}
		kp := strings.Split(v, "=")
		key := kp[0]

		req.Header.Set(key, kp[1])
	}

	//Add header for client
	req.Header.Set(HeaderReceived, webhook.Received)
	req.Header.Set(HeaderSource, source.SourceID)

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err.Error())
	}

	if err != nil || resp.StatusCode > 299 || resp.StatusCode < 200 {
		addRetry(subscription.PkID, source.PkID, webhook.PkID, func(subsPK uint32) {
			log.Printf("Unsubscribe %d because of failed retry attempts\n", subsPK)
			err := removeSubscriptionByPK(db, subsPK)
			if err != nil {
				log.Println(err.Error())
			}
		})
	} else if resp.StatusCode == http.StatusTeapot {
		//Unsubscribe
		err := removeSubscription(db, subscription.SubscriptionID)
		if err != nil {
			log.Println(err.Error())
		}
	} else {
		//Successful notification
		removeRetry(subscription.PkID)
		subscription.trigger(db)
	}
}

//Ping subscriber and check for a valid url
func (subscription *Subscription) startValidation(srcID string) {
	<-time.After(5 * time.Second)

	val, err := subscription.validateSubsrciption(srcID)
	if err != nil || !val {
		log.Println("Ping failed")
		removeSubscription(db, subscription.SubscriptionID)
		return
	}

	err = subscriptionSetValidated(db, subscription.SubscriptionID)
	if err != nil {
		log.Println(err.Error())
	} else {
		log.Println("Successfully validated")
	}
}

func (subscription *Subscription) validateSubsrciption(sourceID string) (bool, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	ur, err := url.Parse(subscription.CallbackURL)
	if err != nil {
		return false, err
	}

	ur.Path = path.Join(ur.Path, EPPingClient)
	req, err := http.NewRequest("GET", ur.String(), strings.NewReader(""))
	if err != nil {
		return false, err
	}

	//Set required header
	req.Header.Set(HeaderSource, sourceID)
	req.Header.Set(HeaderSubsID, subscription.SubscriptionID)

	//Do the request
	res, err := client.Do(req)

	if err != nil {
		return false, err
	}

	if res.StatusCode == http.StatusOK {
		return true, nil
	}

	return false, nil
}
