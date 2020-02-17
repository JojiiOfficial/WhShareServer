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
	Server   configServer
	Database configDBstruct
	HTTP     configHTTPstruct
	HTTPS    configTLSStruct
}

type configServer struct {
	BogonAsCallback bool `default:"false"`
	WorkerCount     int  `default:"8"`
	RetryTimes      map[uint8]uint
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
	Enabled       bool   `default:"false"`
	ListenAddress string `default:":443"`
	CertFile      string
	KeyFile       string
}

//Config for HTTP
type configHTTPstruct struct {
	Enabled       bool   `default:"true"`
	ListenAddress string `default:":80"`
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

	s, err := os.Stat(confFile)
	if createMode || err != nil {
		if createMode {
			if s != nil && s.IsDir() {
				log.Fatalln("This name is already taken by a folder")
				return nil, true
			}
			if !strings.HasSuffix(confFile, ".yml") {
				log.Fatalln("The configFile must end with .yml")
				return nil, true
			}
		}
		config = ConfigStruct{
			Server: configServer{
				BogonAsCallback: false,
				RetryTimes: map[uint8]uint{
					0: 1,
					1: 10,
					2: 30,
					3: 60,
					4: 120,
					5: 10 * 60,
				},
			},
			Database: configDBstruct{
				Host:         "localhost",
				DatabasePort: 3306,
			},
			HTTP: configHTTPstruct{
				Enabled:       true,
				ListenAddress: "127.0.0.1:80",
			},
			HTTPS: configTLSStruct{
				Enabled:       false,
				ListenAddress: ":443",
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
	if !config.HTTP.Enabled && !config.HTTPS.Enabled {
		log.Println("You must at least enable one of the server protocols!")
		return false
	}

	if config.HTTPS.Enabled {
		if len(config.HTTPS.CertFile) == 0 || len(config.HTTPS.KeyFile) == 0 {
			log.Println("If you enable TLS you need to set CertFile and KeyFile!")
			return false
		}
		//Check SSL files
		if !gaw.FileExists(config.HTTPS.CertFile) {
			log.Println("Can't find the SSL certificate. File not found")
			return false
		}
		if !gaw.FileExists(config.HTTPS.KeyFile) {
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
