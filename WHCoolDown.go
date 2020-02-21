package main

import (
	"fmt"
	"time"

	"github.com/muesli/cache2go"

	log "github.com/sirupsen/logrus"
)

type webhookSpamItem struct {
	SourcePKID     uint32
	ReceiveHistory []time.Time
}

//WebhookAntiSpammer an antiSpammer for incoming webhooks
type WebhookAntiSpammer struct {
	//HookReceiveCache cache the IP
	CacheTable    *cache2go.CacheTable
	MaxWebhookAge time.Duration
}

//NewWebhookAntiSpammer constructor for the Webhook anti spam service
func NewWebhookAntiSpammer(tableName string, maxHookAge time.Duration) *WebhookAntiSpammer {
	return &WebhookAntiSpammer{
		CacheTable:    cache2go.Cache(tableName),
		MaxWebhookAge: maxHookAge,
	}
}

//HandleHook handles a hook and returns false if hook should not processed
func (antiSpammer *WebhookAntiSpammer) HandleHook(source *Source) bool {
	hookItem := antiSpammer.addHookToTable(source)

	fmt.Println("Hook history length", len(hookItem.ReceiveHistory))

	return true
}

func (antiSpammer *WebhookAntiSpammer) addHookToTable(source *Source) *webhookSpamItem {
	if antiSpammer.CacheTable.Exists(source.PkID) {
		log.Debug("Is in cache table")

		//Getting webhookItem
		res, _ := antiSpammer.CacheTable.Value(source.PkID)
		hookItem := res.Data().(*webhookSpamItem)

		//Appending current time to receive history
		hookItem.ReceiveHistory = append(hookItem.ReceiveHistory, time.Now())
		return hookItem
	}

	log.Debug("Put into table")
	spamItem := &webhookSpamItem{
		SourcePKID: source.PkID,
		ReceiveHistory: []time.Time{
			time.Now(),
		},
	}

	//Adding new Item to the CacheTable
	antiSpammer.CacheTable.Add(source.PkID, antiSpammer.MaxWebhookAge, spamItem)
	return spamItem
}
