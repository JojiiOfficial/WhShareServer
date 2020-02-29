package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/models"
	"github.com/JojiiOfficial/WhShareServer/services"
)

//Services
var (
	retryService      *services.RetryService      //Handle retries
	cleanService      *services.CleanupService    //Handle old webhooks
	ipRefreshService  *services.IPRefreshService  //Updates external IP
	usageResetService *services.ResetUsageService //Resets user usage each month
	apiService        *services.APIService        //Handle endpoints
)

func startAPI() {
	log.Info("Starting version " + version)

	if config.Server.BogonAsCallback {
		log.Info("Allowing bogon as callbackURL!")
	}
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

	//Create and init retryService
	retryService = services.NewRetryService(db, config)
	retryService.Callback = subCB{retryService: retryService}
	retryService.Start()
	//TODO load retries from DB

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

	//Create the APIService and start it
	apiService = services.NewAPIService(db, config, &ipRefreshService.IP, subCB{retryService: retryService})
	apiService.Start()

	//Startup done
	log.Info("Startup completed")

	//Start loop to tick the services
	go (func() {
		for {
			time.Sleep(time.Hour)

			usageResetService.Tick()
			cleanService.Tick()
			ipRefreshService.Tick()
		}
	})()

	awaitExit(db, apiService)
}

//Shutdown server gracefully
func awaitExit(db *dbhelper.DBhelper, httpServer *services.APIService) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, os.Interrupt, syscall.SIGKILL, syscall.SIGTERM)

	// await os signal
	<-signalChan

	// Create a deadline for the await
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	log.Info("Shutting down server")

	if httpServer.HTTPServer != nil {
		httpServer.HTTPServer.Shutdown(ctx)
		log.Info("HTTP server shutdown complete")
	}

	if httpServer.HTTPTLSServer != nil {
		httpServer.HTTPTLSServer.Shutdown(ctx)
		log.Info("HTTPs server shutdown complete")
	}

	if db != nil && db.DB != nil {
		db.DB.Close()
		log.Info("Database shutdown complete")
	}

	log.Info("Shutting down complete")
	os.Exit(0)
}

//Callbacks for webhooks
type subCB struct {
	retryService *services.RetryService
}

func (subCB subCB) OnSuccess(subscription models.Subscription) {
	if retry, has := subCB.retryService.RetryList[subscription.PkID]; has {
		subCB.retryService.Remove(db, subscription.PkID, retry)
	}

	log.Debug("Removing subscription from retryQueue. Reason: successful notification")
	if !subscription.IsValid {
		subscription.TriggerAndValidate(db)
	} else {
		subscription.Trigger(db)
	}
}

func (subCB subCB) OnError(subscription models.Subscription, source models.Source, webhook models.Webhook) {
	subCB.retryService.Add(db, subscription.PkID, source.PkID, webhook.PkID)
}

func (subCB subCB) OnUnsubscribe(subscription models.Subscription) {
	subscription.Remove(db)
}

func (subCB subCB) OnWebhookReceive(webhook *models.Webhook, source *models.Source) {
	models.NotifyAllSubscriber(db, config, webhook, source, subCB)
}
