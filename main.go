package main

import (
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"gopkg.in/alecthomas/kingpin.v2"
)

const version = "0.20.1a"

var (
	app         = kingpin.New("server", "A Rest server")
	appLogLevel = app.Flag("log-level", "Enable debug mode").HintOptions(LogLevels...).Envar(getEnVar(EnVarLogLevel)).Short('l').Default(LogLevels[2]).String()
	appNoColor  = app.Flag("no-color", "Disable colors").Envar(getEnVar(EnVarNoColor)).Bool()
	appYes      = app.Flag("yes", "Skips confirmations").Short('y').Envar(getEnVar(EnVarYes)).Bool()
	appCfgFile  = app.
			Flag("config", "the configuration file for the subscriber").
			Envar(getEnVar(EnVarConfigFile)).
			Short('c').String()

	//Server commands
	//Server start
	serverCmd      = app.Command("server", "Commands for the server")
	serverCmdStart = serverCmd.Command("start", "Start the server")

	//Config commands
	//Config create
	configCmd           = app.Command("config", "Commands for the config file")
	configCmdCreate     = configCmd.Command("create", "Create config file")
	configCmdCreateName = configCmdCreate.Arg("name", "Config filename").Default(getDefaultConfig()).String()
)

var (
	config *ConfigStruct
	db     *dbhelper.DBhelper
)

func main() {
	//Set app attributes
	app.HelpFlag.Short('h')
	app.Version(version)

	//parsing the args
	parsed := kingpin.MustParse(app.Parse(os.Args[1:]))

	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		TimestampFormat:  time.Stamp,
		FullTimestamp:    true,
		ForceColors:      true,
	})

	if parsed != configCmdCreate.FullCommand() {
		var shouldExit bool
		config, shouldExit = InitConfig(*appCfgFile, false)
		if shouldExit {
			return
		}

		if !config.Check() {
			log.Info("Exiting")
			return
		}

		var err error
		db, err = connectDB(config)
		if err != nil {
			log.Fatalln(err.Error())
			return
		}

	}

	log.Infof("LogLevel: %s\n", *appLogLevel)

	//set app logLevel
	switch *appLogLevel {
	case LogLevels[0]:
		//Debug
		log.SetLevel(log.DebugLevel)
	case LogLevels[1]:
		//Info
		log.SetLevel(log.InfoLevel)
	case LogLevels[2]:
		//Warning
		log.SetLevel(log.WarnLevel)
	case LogLevels[3]:
		//Error
		log.SetLevel(log.ErrorLevel)
	default:
		fmt.Println("LogLevel not found!")
		os.Exit(1)
		return
	}

	switch parsed {
	//Server --------------------
	case serverCmdStart.FullCommand():
		{
			runCmd(config, db)
		}
	//Config --------------------
	case configCmdCreate.FullCommand():
		{
			//whsub config create
			InitConfig(*configCmdCreateName, true)
		}
	}
}
