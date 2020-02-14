package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/mkideal/cli"
	"github.com/thecodeteam/goodbye"
)

type runT struct {
	cli.Helper
}

var runCMD = &cli.Command{
	Name:    "run",
	Aliases: []string{},
	Desc:    "Run the server",
	Argv:    func() interface{} { return new(runT) },
	Fn: func(ct *cli.Context) error {
		//argv := ct.Argv().(*runT)

		ctx := initExitCallback()
		defer goodbye.Exit(ctx, -1)
		config = initConfig(configFile)
		showTimeInLog = config.ShowTimeInLog
		initDB(config)

		isConnected := isConnectedToDB() == nil
		if !isConnected {
			LogError("Couldn't connect to DB!")
			return nil
		}

		router := NewRouter()

		useTLS := checkUseTLS(config)
		if useTLS {
			go (func() {
				if config.TLSPort < 2 {
					LogError("TLS port must be bigger than 1")
					os.Exit(1)
				}
				if config.TLSPort == config.HTTPPort {
					LogCritical("HTTP port can't be the same as TLS port!")
					os.Exit(1)
				}
				tlsprt := strconv.Itoa(config.TLSPort)
				LogInfo("Server started TLS on port (" + tlsprt + ")")
				log.Fatal(http.ListenAndServeTLS(":"+tlsprt, config.CertFile, config.KeyFile, router))
			})()
		}

		if useTLS && config.HTTPPort < 2 {
			for {

			}
		}

		if config.HTTPPort < 2 {
			LogError("HTTP port must be bigger than 1")
			return nil
		}
		httpprt := strconv.Itoa(config.HTTPPort)
		LogInfo("Server started HTTP on port (" + httpprt + ")")
		log.Fatal(http.ListenAndServe(":"+httpprt, router))

		return nil
	},
}

func initExitCallback() context.Context {
	ctx := context.Background()
	goodbye.Notify(ctx)
	goodbye.Register(func(ctx context.Context, sig os.Signal) {
		if db != nil {
			db.Close()
			LogInfo("DB closed")
		}
	})
	return ctx
}

func initConfig(file string) Config {
	return readConfig(file)
}

func checkUseTLS(config Config) (useTLS bool) {
	if len(config.CertFile) > 0 {
		_, err := os.Stat(config.CertFile)
		if err != nil {
			LogError("Certfile not found. HTTP only!")
			return false
		}
		useTLS = true
	}

	if len(config.KeyFile) > 0 {
		_, err := os.Stat(config.KeyFile)
		if err != nil {
			LogError("Keyfile not found. HTTP only!")
			return false
		}
		useTLS = true
	}

	return
}
