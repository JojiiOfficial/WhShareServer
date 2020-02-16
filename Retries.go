package main

import (
	"fmt"
	"log"
	"time"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

//Retry retries after some time
type Retry struct {
	TryNr     uint8
	NextRetry int64
	SourcePK  uint32
	WebhookPK uint32
}

//RetryList list of retries
var RetryList = map[uint32]Retry{}

//RetryTimes constant map of
var RetryTimes = map[uint8]time.Duration{
	0: 1 * time.Minute,
	1: 1 * time.Minute,
	2: 10 * time.Minute,
	3: 30 * time.Minute,
	4: 60 * time.Minute,
	5: 3 * time.Hour,
}

func getNewRetryTime(index uint8) time.Time {
	return time.Now().Add(RetryTimes[index])
}

func addRetry(subscriptionPK, sourcePK, WebhookPK uint32, removeSubs func(uint32)) {
	rl, ok := RetryList[subscriptionPK]
	if ok {
		if rl.TryNr > 5 {
			log.Printf("Removing from %d retry\n", subscriptionPK)
			removeRetry(subscriptionPK)
			removeSubs(subscriptionPK)
			return
		}
		rl.TryNr = rl.TryNr + 1
		rl.NextRetry = getNewRetryTime(rl.TryNr).Unix()

		log.Println("Next retry:", getNewRetryTime(rl.TryNr).Format(time.Stamp))
	} else {
		log.Println("add new retry to list")
		RetryList[subscriptionPK] = Retry{
			WebhookPK: WebhookPK,
			SourcePK:  sourcePK,
			TryNr:     0,
			NextRetry: getNewRetryTime(0).Unix(),
		}
	}
}

func removeRetry(subscriptionPK uint32) {
	if _, ok := RetryList[subscriptionPK]; ok {
		log.Println("removing subscription")
		delete(RetryList, subscriptionPK)
	}
}

func handleRetries(db *dbhelper.DBhelper) {
	for subsPK, retry := range RetryList {
		if retry.NextRetry <= time.Now().Unix() {
			doRetry(db, subsPK, retry.SourcePK, retry.WebhookPK)
		}
	}
}

func doRetry(db *dbhelper.DBhelper, subscriptionPK, sourcePK, WebhookPK uint32) {
	subscription, err := getSubscriptionFromPK(db, subscriptionPK)
	if err != nil {
		log.Println("getSubsFromPK", err.Error())
		return
	}
	source, err := getSourceFromPK(db, sourcePK)
	if err != nil {
		log.Println("getSourceFromPK", err.Error())
		return
	}
	webhook, err := getWebhookFromPK(db, WebhookPK)
	if err != nil {
		log.Println("getWebhookFromPK", err.Error())
		return
	}
	fmt.Println("doing retry")
	go doRequest(*subscription, webhook, source)
}

func startRetryLoop(db *dbhelper.DBhelper) {
	go (func(dbs *dbhelper.DBhelper) {
		for {
			time.Sleep(30 * time.Second)
			handleRetries(dbs)
		}
	})(db)
}
