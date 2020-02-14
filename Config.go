package main

import (
	"log"
	"os"
	"path"
	"strings"

	gaw "github.com/JojiiOfficial/GoAw"
	"github.com/JojiiOfficial/configor"
)

//ConfigStruct config for the server
type ConfigStruct struct {
	Database      configDBstruct
	HTTP          configHTTPstruct
	TLS           configTLSStruct
	ShowTimeInLog bool `default:"true"`
}

type configDBstruct struct {
	Host         string `required:"true"`
	Username     string `required:"true"`
	Database     string `required:"true"`
	Pass         string `required:"true"`
	DatabasePort int    `required:"true" default:"3306"`
}

//Config for HTTPS
type configTLSStruct struct {
	Enabled       bool `default:"false"`
	CertFile      string
	KeyFile       string
	ListenAddress string `default:":"`
	Port          int    `default:"443"`
}

//Config for HTTP
type configHTTPstruct struct {
	Enabled       bool   `default:"true"`
	ListenAddress string `default:":"`
	Port          int    `default:"80"`
}

func getDefaultConfig() string {
	return path.Join(DataDir, DefaultConfigFile)
}

//InitConfig inits the config
//Returns true if system should exit
func InitConfig(confFile string, createMode bool) (*ConfigStruct, bool) {
	var config ConfigStruct
	if len(confFile) == 0 {
		confFile = getDefaultConfig()
	}

	if createMode {
		s, err := os.Stat(confFile)
		if err == nil {
			log.Fatalln("This config already exists!")
			return nil, true
		}
		if s != nil && s.IsDir() {
			log.Fatalln("This name is already taken by a folder")
			return nil, true
		}
		if !strings.HasSuffix(confFile, ".yml") {
			log.Fatalln("The configfile must end with .yml")
			return nil, true
		}
		config = ConfigStruct{
			Database: configDBstruct{
				Host:         "localhost",
				DatabasePort: 3306,
			},
			HTTP: configHTTPstruct{
				Enabled:       true,
				ListenAddress: "127.0.0.1:",
				Port:          80,
			},
			TLS: configTLSStruct{
				Enabled:       false,
				ListenAddress: ":",
				Port:          443,
			},
		}
	}

	isDefault, err := configor.SetupConfig(&config, confFile, configor.NoChange)
	if err != nil {
		log.Fatalln(err.Error())
		return nil, true
	}
	if isDefault {
		log.Println("New config created.")
		if createMode {
			log.Println("Exiting")
			return nil, true
		}
	}

	if err = configor.Load(&config, confFile); err != nil {
		log.Fatalln(err.Error())
		return nil, true
	}

	return &config, false
}

//Check check the config file of logical errors
func (config *ConfigStruct) Check() bool {
	if !config.HTTP.Enabled && !config.TLS.Enabled {
		log.Println("You must at least enable one of the server protocols!")
		return false
	}

	if config.TLS.Enabled {
		if len(config.TLS.CertFile) == 0 || len(config.TLS.KeyFile) == 0 {
			log.Println("If you enable TLS you need to set CertFile and KeyFile!")
			return false
		}
		//Check SSL files
		if !gaw.FileExists(config.TLS.CertFile) {
			log.Println("Can't find the SSL certificate. File not found")
			return false
		}
		if !gaw.FileExists(config.TLS.KeyFile) {
			log.Println("Can't find the SSL key. File not found")
			return false
		}
	}

	if config.Database.DatabasePort < 1 || config.Database.DatabasePort > 65535 {
		log.Printf("Invalid port for database %d\n", config.Database.DatabasePort)
		return false
	}

	return true
}
