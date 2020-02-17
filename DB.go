package main

import (
	"log"
	"strconv"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	_ "github.com/go-sql-driver/mysql"
)

func connectDB(config *ConfigStruct) (*dbhelper.DBhelper, error) {
	log.Println("Connecting to DB")
	db, err := dbhelper.NewDBHelper(dbhelper.Mysql).Open(
		config.Database.Username,
		config.Database.Pass,
		config.Database.Host,
		strconv.Itoa(config.Database.DatabasePort),
		config.Database.Database,
	)
	if err != nil {
		return nil, err
	}
	log.Println("Connected successfully")
	db.Options.Debug = *appDebug
	db.Options.UseColors = !(*appNoColor)
	return db, updateDB(db)
}

func updateDB(db *dbhelper.DBhelper) error {
	db.AddQueryChain(getInitSQL())
	return db.RunUpdate()
}
