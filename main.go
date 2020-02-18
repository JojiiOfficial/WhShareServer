package main

import (
	"log"
	"os"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"gopkg.in/alecthomas/kingpin.v2"
)

const version = "0.19.1a"

var (
	app        = kingpin.New("server", "A Rest server")
	appDebug   = app.Flag("debug", "Enable debug mode").Envar(getEnVar(EnVarDebug)).Short('d').Bool()
	appNoColor = app.Flag("no-color", "Disable colors").Envar(getEnVar(EnVarNoColor)).Bool()
	appYes     = app.Flag("yes", "Skips confirmations").Short('y').Envar(getEnVar(EnVarYes)).Bool()
	appCfgFile = app.
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

	if parsed != configCmdCreate.FullCommand() {
		var shouldExit bool
		config, shouldExit = InitConfig(*appCfgFile, false)
		if shouldExit {
			return
		}

		if !config.Check() {
			if *appDebug {
				log.Println("Exiting")
			}
			return
		}

		var err error
		db, err = connectDB(config)
		if err != nil {
			log.Fatalln(err.Error())
			return
		}
	}

	switch parsed {
	//Server --------------------
	case serverCmdStart.FullCommand():
		{
			runCmd(config, db, *appDebug)
		}
	//Config --------------------
	case configCmdCreate.FullCommand():
		{
			//whsub config create
			InitConfig(*configCmdCreateName, true)
		}
	}
}
