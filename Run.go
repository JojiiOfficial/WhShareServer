package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/mkideal/cli"
	"github.com/thecodeteam/goodbye"
)

type runT struct {
	cli.Helper
}

func runCmd(config *ConfigStruct, dab *dbhelper.DBhelper, debug bool) {
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

	startRetryLoop()
	for {
		time.Sleep(time.Hour)
	}
}

func initExitCallback(db *dbhelper.DBhelper) context.Context {
	ctx := context.Background()
	goodbye.Notify(ctx)
	goodbye.Register(func(ctx context.Context, sig os.Signal) {
		if db.DB != nil {
			db.DB.Close()
			LogInfo("DB closed")
		}
	})
	return ctx
}
