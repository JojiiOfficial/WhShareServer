package main

import (
	"log"
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
	subscriptions, err := getSubscriptionsFromSource(db, source.PkID)
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

	for {
		if pos >= len(subscriptions) {
			break
		}
		read := <-c
		for i := 0; i < read; i++ {
			go startNotfy(&c, pos, &subscriptions[pos], webhook, source)
			pos++
		}
	}
}

func startNotfy(c *chan int, pos int, subscription *Subscription, webhook *Webhook, source *Source) {
	<-time.After(5 * time.Millisecond)
	*c <- 1
}
