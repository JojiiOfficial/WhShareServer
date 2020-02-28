package services

import (
	"time"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/models"
	log "github.com/sirupsen/logrus"
)

//RetryService handles retries
type RetryService struct {
	//RetryList list of retries
	RetryList map[uint32]*models.Retry

	//RetryTimes constant map of
	RetryTimes map[uint8]time.Duration

	db *dbhelper.DBhelper

	handlerInterval time.Duration
	Callback        models.NotifyCallback
}

//NewRetryService create new retryService
func NewRetryService(db *dbhelper.DBhelper, conf *models.ConfigStruct) *RetryService {
	return &RetryService{
		RetryList:       make(map[uint32]*models.Retry),
		RetryTimes:      conf.Server.Retries.RetryTimes,
		handlerInterval: conf.Server.Retries.RetryInterval,
		db:              db,
	}
}

//Add adds a subscription to the retryService
func (retryService *RetryService) Add(db *dbhelper.DBhelper, subscriptionPK, sourcePK, WebhookPK uint32) {
	if _, ok := retryService.RetryList[subscriptionPK]; ok {
		return
	}

	retry, err := models.NewRetry(db, sourcePK, WebhookPK, retryService.getRetryTime(0))
	if err != nil {
		log.Error("Error inserting retry. This retry might not be delivered on an app crash")
	}

	retryService.RetryList[subscriptionPK] = retry

	log.Debug("Add new retry to list. Next retry: ", retry.NextRetry.Format(time.Stamp))
}

//Remove removes a subscription from the retryService
func (retryService *RetryService) Remove(db *dbhelper.DBhelper, subscriptionPK uint32, retry *models.Retry) {
	delete(retryService.RetryList, subscriptionPK)
	retry.Delete(db)
}

//Start starts the retryService
func (retryService *RetryService) Start() {
	go (func() {
		for {
			time.Sleep(retryService.handlerInterval)
			retryService.handle()
		}
	})()
}

func (retryService *RetryService) handle() {
	for subsPK, retry := range retryService.RetryList {
		//If retry time is come
		if retry.NextRetry.Unix() <= time.Now().Unix() {
			if retry.TryNr >= uint8(len(retryService.RetryTimes)) {
				log.Info("Removing subscription. Reason: too many retries")
				err := models.RemoveSubscriptionByPK(retryService.db, subsPK)
				if err != nil {
					log.Println(err.Error())
				}

				retryService.Remove(retryService.db, subsPK, retry)
			} else {
				retry.TryNr++
				retryService.calcNextRetryTime(retry)
				retry.UpdateNext(retryService.db)
				retryService.do(subsPK, retry)
			}
		}
	}
}

func (retryService *RetryService) do(subsPK uint32, retry *models.Retry) {
	subscription, err := models.GetSubscriptionByPK(retryService.db, subsPK)
	if err != nil {
		log.Error("getSubsFromPK", err.Error())
		return
	}
	source, err := models.GetSourceByPK(retryService.db, retry.SourcePK)
	if err != nil {
		log.Error("getSourceFromPK", err.Error())
		return
	}
	webhook, err := models.GetWebhookByPK(retryService.db, retry.WebhookPK)
	if err != nil {
		log.Error("getWebhookFromPK", err.Error())
		return
	}

	log.Debug("Doing retry")

	go subscription.Notify(retryService.db, webhook, source, retryService.Callback)
}

func (retryService *RetryService) calcNextRetryTime(retry *models.Retry) {
	retry.NextRetry = retryService.getRetryTime(retry.TryNr)
}

func (retryService RetryService) getRetryTime(tryNr uint8) time.Time {
	return time.Now().Add(retryService.RetryTimes[tryNr])
}
