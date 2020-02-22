package main

import (
	"math/rand"
	"net/http"
	"strings"
	"time"

	gaw "github.com/JojiiOfficial/GoAw"
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
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

//Notify all subscriber for a given webhook
func notifyAllSubscriber(db *dbhelper.DBhelper, webhook *Webhook, source *Source) {
	subscriptions, err := source.getSubscriptions(db)
	if LogError(err) {
		return
	}

	if len(subscriptions) > 0 {
		log.Debugf("Starting pool for %d subscriber\n", len(subscriptions))

		go (func() {
			startPool(db, webhook, source, subscriptions)
		})()
	} else {
		log.Info("No subscriber found!")
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
	LogError(err)

	if err != nil || resp.StatusCode > 299 || resp.StatusCode < 200 {
		retryService.add(subscription.PkID, source.PkID, webhook.PkID)
	} else if resp.StatusCode == http.StatusTeapot {
		//Unsubscribe
		subscription.remove(db, retryService)
	} else {
		//Successful notification
		retryService.remove(subscription.PkID)
		log.Debug("Removing subscription from retryQueue. Reason: successful notification")
		if !subscription.IsValid {
			subscription.triggerAndValidate(db)
		} else {
			subscription.trigger(db)
		}
	}
}

// ------------------------ Queries

func removeSubscriptionByPK(db *dbhelper.DBhelper, pk uint32, rService RetryService) error {
	_, err := db.Execf("DELETE FROM %s WHERE pk_id=?", []string{TableSubscriptions}, pk)
	rService.remove(pk)
	return err
}

func (subscription Subscription) remove(db *dbhelper.DBhelper, rService *RetryService) error {
	rService.remove(subscription.PkID)
	_, err := db.Execf("DELETE FROM %s WHERE pk_id=?", []string{TableSubscriptions}, subscription.PkID)
	return err
}

func getSubscriptionBySubsID(db *dbhelper.DBhelper, subscriptionID string) (*Subscription, error) {
	var subscription Subscription
	err := db.QueryRowf(&subscription, "SELECT * FROM %s WHERE subscriptionID=? LIMIT 1", []string{TableSubscriptions}, subscriptionID)
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

func getSubscriptionByPK(db *dbhelper.DBhelper, pkID uint32) (*Subscription, error) {
	var subscription Subscription
	err := db.QueryRowf(&subscription, "SELECT * FROM %s WHERE pk_id=? LIMIT 1", []string{TableSubscriptions}, pkID)
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

func (subscription *Subscription) triggerAndValidate(db *dbhelper.DBhelper) error {
	_, err := db.Execf("UPDATE %s SET isValid=1, lastTrigger=now() WHERE subscriptionID=?", []string{TableSubscriptions}, subscription.SubscriptionID)
	return err
}

func (subscription *Subscription) trigger(db *dbhelper.DBhelper) {
	db.Execf("UPDATE %s SET lastTrigger=now() WHERE pk_id=?", []string{TableSubscriptions}, subscription.PkID)
}

func (subscription *Subscription) updateCallback(db *dbhelper.DBhelper, newCallback string) error {
	_, err := db.Execf("UPDATE %s SET callbackURL=? WHERE subscriptionID=?", []string{TableSubscriptions}, newCallback, subscription.SubscriptionID)
	return err
}

func (subscription *Subscription) insert(db *dbhelper.DBhelper) error {
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

func (user *User) getSubscriptionCount(db *dbhelper.DBhelper) (uint32, error) {
	var c uint32
	err := db.QueryRowf(&c, "SELECT COUNT(*) FROM %s WHERE subscriber=?", []string{TableSubscriptions}, user.Pkid)
	return c, err
}
