package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/models"
	"github.com/JojiiOfficial/WhShareServer/services"
	"github.com/thecodeteam/goodbye"
)

var (
	retryService *services.RetryService
)

var currIP string

func runCmd(config *models.ConfigStruct, dab *dbhelper.DBhelper) {
	log.Info("Starting version " + version)

	if config.Server.BogonAsCallback {
		log.Info("Allowing bogon as callbackURL!")
	}

	//initializing exit callback
	ctx := initExitCallback(dab)
	defer goodbye.Exit(ctx, -1)

	//creating new router
	router := NewRouter()

	//Setting up database
	db = dab
	db.SetErrHook(func(err error, query, prefix string) {
		logMessage := prefix + query + ": " + err.Error()

		//Warn only on production
		if isDebug {
			log.Error(logMessage)
		} else {
			log.Warn(logMessage)
		}
	}, dbhelper.ErrHookOptions{
		Prefix:         "Query: ",
		ReturnNilOnErr: false,
	})

	c := make(chan string, 1)
	go (func() {
		c <- getOwnIP()
	})()

	//Starting services

	//Create and start retryService
	retryService = services.NewRetryService(db, config, subCB{retryService: retryService})
	retryService.Start()

	//Start webhook cleaner
	startWebhookCleaner(db)

	currIP = <-c
	if !isIPv4(currIP) {
		log.Fatalf("Error validating IP address! '%s' Exiting\n", currIP)
	} else {
		log.Debugf("Servers IP address is '%s'\n", currIP)
	}

	//Start the WebServer
	startWebServer(router, config)

	log.Info("Startup completed")

	for {
		resetUsageService(db)
		time.Sleep(time.Hour)
		updateCurrIP()
	}
}

//Callback for notifications
type subCB struct {
	retryService *services.RetryService
}

func (subCB subCB) OnSuccess(subscription models.Subscription) {
	subCB.retryService.Remove(subscription.PkID)
	log.Debug("Removing subscription from retryQueue. Reason: successful notification")
	if !subscription.IsValid {
		subscription.TriggerAndValidate(db)
	} else {
		subscription.Trigger(db)
	}
}

func (subCB subCB) OnError(subscription models.Subscription, source models.Source, webhook models.Webhook) {
	subCB.retryService.Add(subscription.PkID, source.PkID, webhook.PkID)
}

func (subCB subCB) OnUnsubscribe(subscription models.Subscription) {
	subscription.Remove(db)
}

func resetUsageService(db *dbhelper.DBhelper) {
	start := time.Now()
	n, err := resetUserResourceUsage(db)
	if err == nil && n > 0 {
		dur := time.Now().Sub(start).String()
		log.Debugf("Resource usage resetting took %s\n", dur)
		log.Infof("Reset resource usage for %d user(s)", n)
	}
}

func updateCurrIP() {
	//Update IP address every hour
	cip := getOwnIP()
	if cip != currIP && isIPv4(cip) {
		log.Infof("Server got new IP address %s\n", cip)
		currIP = cip
	}
}

func startWebServer(router *mux.Router, config *models.ConfigStruct) {
	//Start HTTPS if enabled
	if config.Webserver.HTTPS.Enabled {
		log.Infof("Server started TLS on port (%s)\n", config.Webserver.HTTPS.ListenAddress)
		go (func() {
			log.Fatal(http.ListenAndServeTLS(config.Webserver.HTTPS.ListenAddress, config.Webserver.HTTPS.CertFile, config.Webserver.HTTPS.KeyFile, router))
		})()
	}

	//Start HTTP if enabled
	if config.Webserver.HTTP.Enabled {
		log.Infof("Server started HTTP on port (%s)\n", config.Webserver.HTTP.ListenAddress)
		go (func() {
			log.Fatal(http.ListenAndServe(config.Webserver.HTTP.ListenAddress, router))
		})()
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
