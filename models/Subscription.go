package models

import (
	"math/rand"
	"net/http"
	"strings"
	"time"

	gaw "github.com/JojiiOfficial/GoAw"
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/constants"
	log "github.com/sirupsen/logrus"
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

//TableSubscriptions the tableName for subscriptions
const (
	TableSubscriptions = "Subscriptions"
	TableModes         = "Modes"
)

//NotifyAllSubscriber for a given webhook
func NotifyAllSubscriber(db *dbhelper.DBhelper, config *ConfigStruct, webhook *Webhook, source *Source, callback NotifyCallback) {
	subscriptions, err := source.getSubscriptions(db)
	if LogError(err) {
		return
	}

	if len(subscriptions) > 0 {
		log.Debugf("Starting pool for %d subscriber\n", len(subscriptions))

		go (func() {
			startPool(db, config, webhook, source, subscriptions, callback)
		})()
	} else {
		log.Info("No subscriber found!")
	}
}

//Start notifier pool
func startPool(db *dbhelper.DBhelper, config *ConfigStruct, webhook *Webhook, source *Source, subscriptions []Subscription, callback NotifyCallback) {
	pos := 0

	c := make(chan int, 1)
	c <- config.Server.WorkerCount

	for pos < len(subscriptions) {
		read := <-c
		for i := 0; i < read && pos < len(subscriptions); i++ {

			go (func(c *chan int, subscription *Subscription, webhook *Webhook, source *Source) {
				rand.Seed(time.Now().UnixNano())
				<-time.After(time.Duration(rand.Intn(999)+1) * time.Millisecond)
				subscription.Notify(db, webhook, source, callback)
				*c <- 1
			})(&c, &subscriptions[pos], webhook, source)

			pos++
		}
	}
}

//Notify subscriber
func (subscription *Subscription) Notify(db *dbhelper.DBhelper, webhook *Webhook, source *Source, callback NotifyCallback) (*http.Response, error) {
	client := &http.Client{
		Timeout: 20 * time.Second,
	}
	req, _ := http.NewRequest("POST", subscription.CallbackURL, strings.NewReader(webhook.Payload))

	//Load headers from webhook.Headers
	setHeadersFromStr(webhook.Headers, &req.Header)

	//Add header for client
	req.Header.Set(constants.HeaderReceived, webhook.Received)
	req.Header.Set(constants.HeaderSource, source.SourceID)
	req.Header.Set(constants.HeaderSubsID, subscription.SubscriptionID)

	//Do the request
	resp, err := client.Do(req)
	LogError(err)

	if err != nil || resp.StatusCode > 299 || resp.StatusCode < 200 {
		callback.OnError(*subscription, *source, *webhook)
	} else if resp.StatusCode == http.StatusTeapot {
		//Unsubscribe
		callback.OnUnsubscribe(*subscription)
	} else {
		//Successful notification
		callback.OnSuccess(*subscription)
	}

	return resp, err
}

// ------------------------ Queries

//RemoveSubscriptionByPK removes a subscription by pk
func RemoveSubscriptionByPK(db *dbhelper.DBhelper, pk uint32) error {
	_, err := db.Execf("DELETE FROM %s WHERE pk_id=?", []string{TableSubscriptions}, pk)
	return err
}

//Remove removes/unsubscribes to a subscription
func (subscription Subscription) Remove(db *dbhelper.DBhelper) error {
	_, err := db.Execf("DELETE FROM %s WHERE pk_id=?", []string{TableSubscriptions}, subscription.PkID)
	return err
}

//GetSubscriptionBySubsID get the subscription by subscriptionID
func GetSubscriptionBySubsID(db *dbhelper.DBhelper, subscriptionID string) (*Subscription, error) {
	var subscription Subscription
	err := db.QueryRowf(&subscription, "SELECT * FROM %s WHERE subscriptionID=? LIMIT 1", []string{TableSubscriptions}, subscriptionID)
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

//GetSubscriptionByPK get subscription by pk
func GetSubscriptionByPK(db *dbhelper.DBhelper, pkID uint32) (*Subscription, error) {
	var subscription Subscription
	err := db.QueryRowf(&subscription, "SELECT * FROM %s WHERE pk_id=? LIMIT 1", []string{TableSubscriptions}, pkID)
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

//TriggerAndValidate triggers the subscription and set validate=1
func (subscription *Subscription) TriggerAndValidate(db *dbhelper.DBhelper) error {
	_, err := db.Execf("UPDATE %s SET isValid=1, lastTrigger=now() WHERE subscriptionID=?", []string{TableSubscriptions}, subscription.SubscriptionID)
	return err
}

//Trigger the subscription
func (subscription *Subscription) Trigger(db *dbhelper.DBhelper) {
	db.Execf("UPDATE %s SET lastTrigger=now() WHERE pk_id=?", []string{TableSubscriptions}, subscription.PkID)
}

//UpdateCallback updates the callback for a subscription
func (subscription *Subscription) UpdateCallback(db *dbhelper.DBhelper, newCallback string) error {
	_, err := db.Execf("UPDATE %s SET callbackURL=? WHERE subscriptionID=?", []string{TableSubscriptions}, newCallback, subscription.SubscriptionID)
	return err
}

//Insert inserts the subscription into the db
func (subscription *Subscription) Insert(db *dbhelper.DBhelper) error {
	subscription.SubscriptionID = gaw.RandString(32)
	rs, err := db.Execf("INSERT INTO %s (subscriptionID, subscriber, source, callbackURL) VALUES (?,?,?,?)", []string{TableSubscriptions}, subscription.SubscriptionID, subscription.UserID, subscription.Source, subscription.CallbackURL)
	if err != nil {
		return err
	}
	id, err := rs.LastInsertId()
	if err != nil {
		return err
	}
	subscription.PkID = uint32(id)
	return nil
}

func (source *Source) getSubscriptions(db *dbhelper.DBhelper) ([]Subscription, error) {
	var subscriptions []Subscription
	err := db.QueryRowsf(&subscriptions, "SELECT * FROM %s WHERE source=?", []string{TableSubscriptions}, source.PkID)
	return subscriptions, err
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

// SubscriptionExists check if a subscription exists
func SubscriptionExists(db *dbhelper.DBhelper, sourceID uint32, callbackURL string) (bool, error) {
	var c int
	err := db.QueryRowf(&c, "SELECT COUNT(*) FROM %s WHERE source=? AND callbackURL=?", []string{TableSubscriptions}, sourceID, callbackURL)
	if err != nil {
		return false, err
	}
	return c > 0, nil
}
