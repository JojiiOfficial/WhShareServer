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
func NewRetryService(db *dbhelper.DBhelper, conf *models.ConfigStruct) *RetryService {
	return &RetryService{
		RetryList:       make(map[uint32]*Retry),
		RetryTimes:      conf.Server.Retries.RetryTimes,
		handlerInterval: conf.Server.Retries.RetryInterval,
		db:              db,
	}
}

func (retryService *RetryService) calcNextRetryTime(retry *Retry) {
	retry.NextRetry = time.Now().Add(retryService.RetryTimes[retry.TryNr])
}

//Add adds a subscription to the retryService
func (retryService *RetryService) Add(subscriptionPK, sourcePK, WebhookPK uint32) {
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

	log.Debug("Add new retry to list. Next retry:", retry.NextRetry.Format(time.Stamp))
}

//Remove removes a subscription from the retryService
func (retryService *RetryService) Remove(subscriptionPK uint32) {
	if _, ok := retryService.RetryList[subscriptionPK]; ok {
		delete(retryService.RetryList, subscriptionPK)
	}
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
			} else {
				retry.TryNr++
				retryService.calcNextRetryTime(retry)
				retry.do(subsPK, retryService)
			}
		}
	}
}

func (retry *Retry) do(subsPK uint32, retryService *RetryService) {
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

	go subscription.Notify(retryService.db, webhook, source)
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

//LogError returns true on error
func LogError(err error, context ...map[string]interface{}) bool {
	if err == nil {
		return false
	}

	if len(context) > 0 {
		log.WithFields(context[0]).Error(err.Error())
	} else {
		log.Error(err.Error())
	}
	return true
}
