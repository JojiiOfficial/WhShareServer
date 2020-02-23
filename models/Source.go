package models

import (
	gaw "github.com/JojiiOfficial/GoAw"
	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

//TODO clean up

//Source a webhook source
type Source struct {
	PkID         uint32 `db:"pk_id" orm:"pk,ai" json:"-"`
	Name         string `db:"name" json:"name"`
	SourceID     string `db:"sourceID" json:"sourceID"`
	Description  string `db:"description" json:"description"`
	Secret       string `db:"secret" json:"secret"`
	CreatorID    uint32 `db:"creator" json:"-"`
	CreationTime string `db:"creationTime" json:"crTime"`
	IsPrivate    bool   `db:"private" json:"isPrivate"`
	Mode         uint8  `db:"mode" json:"mode"`
	Creator      User   `db:"-" json:"-"`
}

//TableSources the db tableName for sources
const TableSources = "Sources"

//GetSourceFromSourceID get source from sourceID
func GetSourceFromSourceID(db *dbhelper.DBhelper, sourceID string) (*Source, error) {
	var source Source
	err := db.QueryRowf(&source, "SELECT * FROM %s WHERE sourceID=? LIMIT 1", []string{TableSources}, sourceID)
	if err != nil {
		return nil, err
	}
	return &source, nil
}

//GetSourcesForUser gets sources for user
func GetSourcesForUser(db *dbhelper.DBhelper, userID uint32) ([]Source, error) {
	var sources []Source
	err := db.QueryRowsf(&sources, "SELECT * FROM %s WHERE creator=?", []string{TableSources}, userID)
	if err != nil {
		return nil, err
	}
	return sources, nil
}

//GetSourceByPK gets source by pk
func GetSourceByPK(db *dbhelper.DBhelper, sourceID uint32) (*Source, error) {
	var source Source
	err := db.QueryRowf(&source, "SELECT * FROM %s WHERE pk_id=? LIMIT 1", []string{TableSources}, sourceID)
	if err != nil {
		return nil, err
	}
	return &source, nil
}

//Insert inserts source into DB
func (source *Source) Insert(db *dbhelper.DBhelper) error {
	secret := gaw.RandString(48)
	sid := gaw.RandString(32)
	rs, err := db.Execf("INSERT INTO %s (creator, name, description, secret, private, sourceID, mode) VALUES(?,?,?,?,?,?,?)", []string{TableSources}, source.Creator.Pkid, source.Name, source.Description, secret, source.IsPrivate, sid, source.Mode)
	if err != nil {
		return err
	}
	id, err := rs.LastInsertId()
	if err != nil {
		return err
	}
	source.PkID = uint32(id)
	source.Secret = secret
	source.SourceID = sid

	return nil
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
	_, err := db.Execf("DELETE FROM %s WHERE sourceID=?", []string{TableWebhooks}, source.PkID)
	if err != nil {
		return err
	}
	_, err = db.Execf("DELETE FROM %s WHERE source=?", []string{TableSubscriptions}, source.PkID)
	if err != nil {
		return err
	}
	_, err = db.Execf("DELETE FROM %s WHERE pk_id=?", []string{TableSources}, source.PkID)
	return err
}
