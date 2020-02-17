package main

import (
	"log"
	"time"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

//RetryService handles retries
type RetryService struct {
	//RetryList list of retries
	RetryList map[uint32]*Retry

	//RetryTimes constant map of
	RetryTimes map[uint8]time.Duration

	db *dbhelper.DBhelper

	handlerInterval time.Duration
}

//Retry retries after some time
type Retry struct {
	TryNr     uint8
	NextRetry time.Time
	SourcePK  uint32
	WebhookPK uint32
}

//NewRetryService create new retryService
func NewRetryService(db *dbhelper.DBhelper, conf *ConfigStruct) *RetryService {
	return &RetryService{
		RetryList:       make(map[uint32]*Retry),
		RetryTimes:      config.Server.Retries.RetryTimes,
		handlerInterval: config.Server.Retries.RetryInterval,
		db:              db,
	}
}

func (retryService *RetryService) calcNextRetryTime(retry *Retry) {
	retry.NextRetry = time.Now().Add(retryService.RetryTimes[retry.TryNr])
}

func (retryService *RetryService) add(subscriptionPK, sourcePK, WebhookPK uint32) {
	if _, ok := retryService.RetryList[subscriptionPK]; ok {
		return
	}

	retry := &Retry{
		WebhookPK: WebhookPK,
		SourcePK:  sourcePK,
		TryNr:     0,
	}
	retryService.calcNextRetryTime(retry)
	retryService.RetryList[subscriptionPK] = retry

	log.Println("add new retry to list. Next retry:", retry.NextRetry.Format(time.Stamp))
}

func (retryService *RetryService) remove(subscriptionPK uint32) {
	if _, ok := retryService.RetryList[subscriptionPK]; ok {
		delete(retryService.RetryList, subscriptionPK)
	}
}

func (retryService *RetryService) handle() {
	for subsPK, retry := range retryService.RetryList {
		//If retry time is come
		if retry.NextRetry.Unix() <= time.Now().Unix() {
			if retry.TryNr >= uint8(len(retryService.RetryTimes)) {
				log.Println("Removing subscription. Reason: too many retries")

				retryService.remove(subsPK)
				err := removeSubscriptionByPK(retryService.db, subsPK)
				if err != nil {
					log.Println(err.Error())
				}
			} else {
				retry.TryNr++
				retryService.calcNextRetryTime(retry)
				retry.do(subsPK)
			}
		}
	}
}

func (retry *Retry) do(subsPK uint32) {
	subscription, err := getSubscriptionFromPK(retryService.db, subsPK)
	if err != nil {
		log.Println("getSubsFromPK", err.Error())
		return
	}
	source, err := getSourceFromPK(retryService.db, retry.SourcePK)
	if err != nil {
		log.Println("getSourceFromPK", err.Error())
		return
	}
	webhook, err := getWebhookFromPK(retryService.db, retry.WebhookPK)
	if err != nil {
		log.Println("getWebhookFromPK", err.Error())
		return
	}

	log.Printf("Doing retry")

	go subscription.Notify(webhook, source)
}

func (retryService *RetryService) start() {
	go (func() {
		for {
			time.Sleep(retryService.handlerInterval)
			retryService.handle()
		}
	})()
}
