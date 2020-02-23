package models

import (
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

//TODO clean up

//Webhook the actual webhook from a server
type Webhook struct {
	PkID     uint32 `db:"pk_id" orm:"pk,ai"`
	SourceID uint32 `db:"sourceID"`
	Headers  string `db:"header"`
	Payload  string `db:"payload"`
	Received string `db:"received"`
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
	rs, err := db.Execf("INSERT INTO %s (sourceID, header, payload) VALUES(?,?,?)", []string{TableWebhooks}, webhook.SourceID, webhook.Headers, webhook.Payload)
	if err != nil {
		return err
	}
	id, err := rs.LastInsertId()
	if err != nil {
		return err
	}
	webhook.PkID = uint32(id)
	return nil
}
