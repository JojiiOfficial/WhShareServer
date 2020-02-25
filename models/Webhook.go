package models

import (
	"time"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

//Webhook the actual webhook from a server
type Webhook struct {
	PkID     uint32    `db:"pk_id" orm:"pk,ai"`
	SourceID uint32    `db:"sourceID"`
	Headers  string    `db:"header"`
	Payload  string    `db:"payload"`
	Received time.Time `db:"received"`
}

//TableWebhooks table for the webhooks
const TableWebhooks = "Webhooks"

//GetWebhookByPK returns webhook by giving a webhook pk_id
func GetWebhookByPK(db *dbhelper.DBhelper, webhookID uint32) (*Webhook, error) {
	var webhook Webhook
	err := db.QueryRowf(&webhook, "SELECT * FROM %s WHERE pk_id=? LIMIT 1", []string{TableWebhooks}, webhookID)
	if err != nil {
		return nil, err
	}
	return &webhook, nil
}

//Insert webhook
func (webhook *Webhook) Insert(db *dbhelper.DBhelper) error {
	_, err := db.Insert(webhook, &dbhelper.InsertOption{
		TableName: TableWebhooks,
		SetPK:     true,
	})
	return err
}
