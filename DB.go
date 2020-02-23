package main

import (
	"strconv"

	log "github.com/sirupsen/logrus"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/models"
	_ "github.com/go-sql-driver/mysql"
)

//TODO merge with Queries.go

func connectDB(config *models.ConfigStruct) (*dbhelper.DBhelper, error) {
	log.Debug("Connecting to DB")
	db, err := dbhelper.NewDBHelper(dbhelper.Mysql).Open(
		config.Server.Database.Username,
		config.Server.Database.Pass,
		config.Server.Database.Host,
		strconv.Itoa(config.Server.Database.DatabasePort),
		config.Server.Database.Database,
	)
	if err != nil {
		return nil, err
	}
	log.Info("Connected successfully")

	//Only debugMode if logLevel is debug
	db.Options.Debug = isDebug

	db.Options.UseColors = !(*appNoColor)
	return db, updateDB(db)
}

func updateDB(db *dbhelper.DBhelper) error {
	db.AddQueryChain(getInitSQL())
	return db.RunUpdate()
}
