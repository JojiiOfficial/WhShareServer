package models

import (
	"time"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

//Retry retries after some time
type Retry struct {
	PKid      uint32    `db:"pk_id" orm:"pk,ai"`
	TryNr     uint8     `db:"tryNr"`
	NextRetry time.Time `db:"nextRetry"`
	SourcePK  uint32    `db:"sourcePK"`
	WebhookPK uint32    `db:"webhookPK"`
}

//TableRetries table containing retries
const TableRetries = "Retries"

//NewRetry create now Retry
func NewRetry(db *dbhelper.DBhelper, sourcePK, webhookPk uint32, nextRetryTime time.Time) (*Retry, error) {
	retry := &Retry{
		TryNr:     0,
		SourcePK:  sourcePK,
		WebhookPK: webhookPk,
		NextRetry: nextRetryTime,
	}

	//Insert retry into DB
	err := retry.insert(db)
	if err != nil {
		return nil, err
	}

	return retry, nil
}

func (retry *Retry) insert(db *dbhelper.DBhelper) error {
	_, err := db.Insert(retry, &dbhelper.InsertOption{
		TableName: TableRetries,
		SetPK:     true,
	})

	return err
}

//UpdateNext updates a retry
func (retry *Retry) UpdateNext(db *dbhelper.DBhelper) error {
	_, err := db.Execf("UPDATE %s SET tryNr=?, nextRetry=(FROM_UNIXTIME(?)) WHERE pk_id=?", []string{TableRetries}, retry.TryNr, retry.NextRetry, retry.PKid)
	return err
}

//Delete deletes a retry from DB
func (retry Retry) Delete(db *dbhelper.DBhelper) error {
	_, err := db.Execf("DELETE FROM %s WHERE pk_id=?", []string{TableRetries}, retry.PKid)
	return err
}
