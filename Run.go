package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/thecodeteam/goodbye"
)

func runCmd(config *ConfigStruct, dab *dbhelper.DBhelper, debug bool) {
	log.Println("Starting version " + version)

	if config.BogonAsCallback {
		log.Println("Allowing bogon as callbackURL!")
	}

	ctx := initExitCallback(dab)
	defer goodbye.Exit(ctx, -1)

	router := NewRouter()
	db = dab

	if config.TLS.Enabled {
		go (func() {
			address := config.TLS.ListenAddress + strconv.Itoa(config.TLS.Port)
			if debug {
				log.Printf("Server started TLS on port (%s)\n", address)
			}
			log.Fatal(http.ListenAndServeTLS(address, config.TLS.CertFile, config.TLS.KeyFile, router))
		})()
	}
	if config.HTTP.Enabled {
		go (func() {
			address := config.HTTP.ListenAddress + strconv.Itoa(config.HTTP.Port)
			if debug {
				log.Printf("Server started HTTP on port (%s)\n", address)
			}
			log.Fatal(http.ListenAndServe(address, router))
		})()
	}

	startRetryLoop(db)
	startWebhookCleaner(db)

	for {
		time.Sleep(time.Hour)
	}
}

//Close db connection on exit
func initExitCallback(db *dbhelper.DBhelper) context.Context {
	ctx := context.Background()
	goodbye.Notify(ctx)
	goodbye.Register(func(ctx context.Context, sig os.Signal) {
		if db.DB != nil {
			db.DB.Close()
			log.Println("DB closed")
		}
	})
	return ctx
}

//A goroutine which deletes every hour unused webhooks
func startWebhookCleaner(dba *dbhelper.DBhelper) {
	if *appDebug {
		log.Println("Start cleaner")
	}
	go (func(db *dbhelper.DBhelper) {
		for {
			deleteOldHooks(db)
			time.Sleep(1 * time.Hour)
		}
	})(dba)
}
