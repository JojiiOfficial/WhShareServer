package models

import (
	"time"

	gaw "github.com/JojiiOfficial/GoAw"
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

//Source a webhook source
type Source struct {
	PkID         uint32    `db:"pk_id" orm:"pk,ai" json:"-"`
	Name         string    `db:"name" json:"name"`
	SourceID     string    `db:"sourceID" json:"sourceID"`
	Description  string    `db:"description" json:"description"`
	Secret       string    `db:"secret" json:"secret"`
	CreatorID    uint32    `db:"creator" json:"-"`
	CreationTime time.Time `db:"creationTime" json:"crTime"`
	IsPrivate    bool      `db:"private" json:"isPrivate"`
	Mode         uint8     `db:"mode" json:"mode"`
	Creator      User      `db:"-" orm:"-" json:"-"`
}

//TableSources the db tableName for sources
const TableSources = "Sources"

//GetSourceFromSourceID get source from sourceID
func GetSourceFromSourceID(db *dbhelper.DBhelper, sourceID string) (*Source, error) {
	var source Source
	err := db.WithHook(dbhelper.NoHook).QueryRowf(&source, "SELECT * FROM %s WHERE sourceID=? LIMIT 1", []string{TableSources}, sourceID)
	if err != nil {
		return nil, err
	}
	return &source, nil
}

//GetSourcesForUser get all sources created by the given user
func GetSourcesForUser(db *dbhelper.DBhelper, userID uint32) ([]Source, error) {
	var sources []Source
	err := db.QueryRowsf(&sources, "SELECT * FROM %s WHERE creator=?", []string{TableSources}, userID)
	if err != nil {
		return nil, err
	}
	return sources, nil
}

//GetSourceByPK get source by pk_id
func GetSourceByPK(db *dbhelper.DBhelper, pkID uint32) (*Source, error) {
	var source Source
	err := db.QueryRowf(&source, "SELECT * FROM %s WHERE pk_id=? LIMIT 1", []string{TableSources}, pkID)
	if err != nil {
		return nil, err
	}
	return &source, nil
}

//Insert source into DB
func (source *Source) Insert(db *dbhelper.DBhelper) error {
	source.Secret = gaw.RandString(48)
	source.SourceID = gaw.RandString(32)

	_, err := db.Insert(source, &dbhelper.InsertOption{
		TableName: TableSources,
		SetPK:     true,
	})

	return err
}

//Update source
func (source *Source) Update(db *dbhelper.DBhelper, field, newText string, arg ...bool) error {
	if newText == "-" && len(arg) > 0 {
		newText = "NULL"
	}
	_, err := db.Execf("UPDATE %s SET %s=? WHERE pk_id=?", []string{TableSources, field}, newText, source.PkID)
	return err
}

//Delete source
func (source *Source) Delete(db *dbhelper.DBhelper) error {
	//Delete all retries
	_, err := db.Execf("DELETE FROM %s WHERE sourcePK=?", []string{TableRetries}, source.PkID)
	if err != nil {
		return err
	}

	//Delete all webhooks assigned to this source
	_, err = db.Execf("DELETE FROM %s WHERE sourceID=?", []string{TableWebhooks}, source.PkID)
	if err != nil {
		return err
	}

	//Delete all subscriptions assigned to this source
	_, err = db.Execf("DELETE FROM %s WHERE source=?", []string{TableSubscriptions}, source.PkID)
	if err != nil {
		return err
	}

	//Delete the source
	_, err = db.Execf("DELETE FROM %s WHERE pk_id=?", []string{TableSources}, source.PkID)
	return err
}
