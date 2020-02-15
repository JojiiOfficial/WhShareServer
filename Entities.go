package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

func deleteSource(db *dbhelper.DBhelper, sourceID uint32) error {
	_, err := db.Execf("DELETE FROM %s WHERE source=?", []string{TableSubscriptions}, sourceID)
	if err != nil {
		return err
	}
	_, err = db.Execf("DELETE FROM %s WHERE pk_id=?", []string{TableSources}, sourceID)
	return err
}

func notifySubscriber(db *dbhelper.DBhelper, webhook *Webhook, source *Source) {
	subscriptions, err := getValidSubscriptionsFromSource(db, source.PkID)
	if err != nil {
		log.Println(err.Error())
		return
	}

	go startPool(db, webhook, source, subscriptions)
}

func startPool(db *dbhelper.DBhelper, webhook *Webhook, source *Source, subscriptions []Subscription) {
	const numWorkers = 4

	pos := 0

	c := make(chan int, 1)
	c <- numWorkers

	for pos < len(subscriptions) {
		read := <-c
		for i := 0; i < read && pos < len(subscriptions); i++ {
			fmt.Println("pos", pos)
			go startNotfy(&c, subscriptions[pos], webhook, source)
			pos++
		}
	}
}

func startNotfy(c *chan int, subscription Subscription, webhook *Webhook, source *Source) {
	rand.Seed(time.Now().UnixNano())
	//Wait some milliseconds to circulate the traffic
	<-time.After(time.Duration(rand.Intn(999)+1) * time.Millisecond)

	fmt.Printf("Notifying user %d\n", subscription.UserID)

	doRequest(subscription, webhook, source)

	*c <- 1
}

func doRequest(subscription Subscription, webhook *Webhook, source *Source) {
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
		fmt.Println(err.Error())
	}
	if err != nil || resp.StatusCode > 299 || resp.StatusCode < 200 {
		addRetry(subscription.PkID, source.PkID, webhook.PkID, func(subsPK uint32) {
			fmt.Printf("Unsubscribe %d because of failed retry attempts\n", subsPK)
			err := removeSubscriptionByPK(db, subsPK)
			if err != nil {
				fmt.Println(err.Error())
			}
		})
	} else {
		removeRetry(subscription.PkID)
		subscription.trigger(db)
	}
}

func startValidation(cbURL, srcID, subsID string) {
	<-time.After(10 * time.Second)
	val, err := validateSubsrciption(cbURL, subsID, srcID)
	if err != nil || !val {
		removeSubscription(db, subsID)
		return
	}
	err = subscriptionSetValidated(db, subsID)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Successfully validated")
	}
}

func validateSubsrciption(u, subID, srcID string) (bool, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	ur, err := url.Parse(u)
	if err != nil {
		return false, err
	}
	ur.Path = path.Join(ur.Path, "ping")
	req, err := http.NewRequest("GET", ur.String(), strings.NewReader("b"))
	if err != nil {
		return false, err
	}
	req.Header.Set(HeaderSource, srcID)
	req.Header.Set(HeaderSubsID, subID)

	res, err := client.Do(req)

	if err != nil {
		return false, err
	}

	if res.StatusCode == http.StatusOK {
		return true, nil
	}

	fmt.Println(res.StatusCode)

	return false, nil
}
