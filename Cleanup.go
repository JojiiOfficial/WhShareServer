package main

import (
	"fmt"
	"time"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
)

func startCleaner(dba *dbhelper.DBhelper) {
	if *appDebug {
		fmt.Println("Start cleaner")
	}
	go (func(db *dbhelper.DBhelper) {
		for {
			deleteOldHooks(db)
			time.Sleep(1 * time.Hour)
		}
	})(dba)
}
