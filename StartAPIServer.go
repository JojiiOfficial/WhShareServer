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

//Services
var (
	retryService      *services.RetryService
	cleanService      *services.CleanupService
	ipRefreshService  *services.IPRefreshService
	usageResetService *services.ResetUsageService
)

var (
	currIP string
)

func startAPI() {
	log.Info("Starting version " + version)

	if config.Server.BogonAsCallback {
		log.Info("Allowing bogon as callbackURL!")
	}

	//initializing exit callback
	ctx := initExitCallback(db)
	defer goodbye.Exit(ctx, -1)

	//Setting up database
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

	//creating new router
	router := NewRouter()

	//Create and init retryService
	retryService = services.NewRetryService(db, config)
	retryService.Callback = subCB{retryService: retryService}
	retryService.Start()

	//Create cleanupService
	cleanService = services.NewCleanupService(db)
	//If cleaning fails, exit
	if err := <-cleanService.Tick(); err != nil {
		log.Fatal(err)
	}

	//Create usageResetService and reset the users
	usageResetService = services.NewResetUsageService(db)
	//If resetting user usage fails, exit
	if err := <-usageResetService.Tick(); err != nil {
		log.Fatal(err)
	}

	//Create IPRefreshService
	ipRefreshService = services.NewIPRefreshService(db)
	if !ipRefreshService.Init() {
		log.Fatalf("Error validating IP address! '%s' Exiting\n", ipRefreshService.IP)
		return
	}
	log.Debugf("Servers IP address is '%s'\n", ipRefreshService.IP)

	//Start the WebServer
	startWebServer(router, config)

	log.Info("Startup completed")

	//Start loop to tick the services
	for {
		time.Sleep(time.Hour)

		usageResetService.Tick()
		cleanService.Tick()
		ipRefreshService.Tick()
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
