package main

import (
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

//MaxRetries count of maximum retries
const MaxRetries = 5

//RetryList list of retries
var RetryList = map[uint32]*Retry{}

//RetryTimes constant map of
var RetryTimes = map[uint8]time.Duration{
	0: 1 * time.Minute,
	1: 10 * time.Minute,
	2: 30 * time.Minute,
	3: 60 * time.Minute,
	4: 120 * time.Minute,
	5: 10 * time.Hour,
}

func calcNextRetryTime(index uint8) time.Time {
	return time.Now().Add(RetryTimes[index])
}

func addRetry(subscriptionPK, sourcePK, WebhookPK uint32) {
	rl, ok := RetryList[subscriptionPK]
	if ok {
		rl.TryNr++
		rl.NextRetry = calcNextRetryTime(rl.TryNr).Unix()
		log.Println("Next retry:", calcNextRetryTime(rl.TryNr).Format(time.Stamp))
	} else {
		log.Println("add new retry to list. ", "Next retry:", calcNextRetryTime(0).Format(time.Stamp))
		RetryList[subscriptionPK] = &Retry{
			WebhookPK: WebhookPK,
			SourcePK:  sourcePK,
			TryNr:     0,
			NextRetry: calcNextRetryTime(0).Unix(),
		}
	}
}

func removeRetry(subscriptionPK uint32) {
	if _, ok := RetryList[subscriptionPK]; ok {
		log.Println("Removing subscription. Reason: too many retries")
		delete(RetryList, subscriptionPK)
	}
}

func handleRetries(db *dbhelper.DBhelper) {
	for subsPK, retry := range RetryList {
		//If retry time is come
		if retry.NextRetry <= time.Now().Unix() {
			if retry.TryNr >= MaxRetries {
				removeRetry(subsPK)
				err := removeSubscriptionByPK(db, subsPK)
				if err != nil {
					log.Println(err.Error())
				}
			} else {
				doRetry(db, subsPK, retry)
			}
		}
	}
}

func doRetry(db *dbhelper.DBhelper, subsPK uint32, retry *Retry) {
	subscription, err := getSubscriptionFromPK(db, subsPK)
	if err != nil {
		log.Println("getSubsFromPK", err.Error())
		return
	}
	source, err := getSourceFromPK(db, retry.SourcePK)
	if err != nil {
		log.Println("getSourceFromPK", err.Error())
		return
	}
	webhook, err := getWebhookFromPK(db, retry.WebhookPK)
	if err != nil {
		log.Println("getWebhookFromPK", err.Error())
		return
	}

	log.Printf("Doing retry")

	go subscription.Notify(webhook, source)
}

func startRetryLoop(db *dbhelper.DBhelper) {
	go (func(dbs *dbhelper.DBhelper) {
		for {
			time.Sleep(10 * time.Second)
			handleRetries(dbs)
		}
	})(db)
}
