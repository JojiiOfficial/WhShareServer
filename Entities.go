package main

import (
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
