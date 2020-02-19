package main

import (
	"context"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/thecodeteam/goodbye"
)

var (
	retryService *RetryService
)

func runCmd(config *ConfigStruct, dab *dbhelper.DBhelper) {
	log.Info("Starting version " + version)

	if config.Server.BogonAsCallback {
		log.Info("Allowing bogon as callbackURL!")
	}

	ctx := initExitCallback(dab)
	defer goodbye.Exit(ctx, -1)

	router := NewRouter()
	db = dab

	db.SetErrHook(func(err error, query, prefix string) {
		log.Error(prefix + query)
	}, dbhelper.ErrHookOptions{
		Prefix:         "In query: ",
		ReturnNilOnErr: false,
	})

	if config.Webserver.HTTPS.Enabled {
		go (func() {
			log.Infof("Server started TLS on port (%s)\n", config.Webserver.HTTPS.ListenAddress)
			log.Fatal(http.ListenAndServeTLS(config.Webserver.HTTPS.ListenAddress, config.Webserver.HTTPS.CertFile, config.Webserver.HTTPS.KeyFile, router))
		})()
	}
	if config.Webserver.HTTP.Enabled {
		go (func() {
			log.Infof("Server started HTTP on port (%s)\n", config.Webserver.HTTP.ListenAddress)
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
			if !LogError(db.DB.Close()) {
				log.Info("DB closed")
			}
		}
	})
	return ctx
}

//A goroutine which deletes every hour unused webhooks
func startWebhookCleaner(dba *dbhelper.DBhelper) {
	log.Info("Start cleaner")
	go (func(db *dbhelper.DBhelper) {
		for {
			deleteOldHooks(db)
			time.Sleep(1 * time.Hour)
		}
	})(dba)
}
