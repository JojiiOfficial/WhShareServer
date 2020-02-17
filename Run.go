package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/thecodeteam/goodbye"
)

var (
	retryService *RetryService
)

func runCmd(config *ConfigStruct, dab *dbhelper.DBhelper, debug bool) {
	log.Println("Starting version " + version)

	if config.Server.BogonAsCallback {
		log.Println("Allowing bogon as callbackURL!")
	}

	ctx := initExitCallback(dab)
	defer goodbye.Exit(ctx, -1)

	router := NewRouter()
	db = dab

	if config.Webserver.HTTPS.Enabled {
		go (func() {
			if debug {
				log.Printf("Server started TLS on port (%s)\n", config.Webserver.HTTPS.ListenAddress)
			}
			log.Fatal(http.ListenAndServeTLS(config.Webserver.HTTPS.ListenAddress, config.Webserver.HTTPS.CertFile, config.Webserver.HTTPS.KeyFile, router))
		})()
	}
	if config.Webserver.HTTP.Enabled {
		go (func() {
			if debug {
				log.Printf("Server started HTTP on port (%s)\n", config.Webserver.HTTP.ListenAddress)
			}
			log.Fatal(http.ListenAndServe(config.Webserver.HTTP.ListenAddress, router))
		})()
	}

	retryService = NewRetryService(db, config)
	retryService.start()

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
