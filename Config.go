package main

import (
	"os"
	"path"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	gaw "github.com/JojiiOfficial/GoAw"
	"github.com/JojiiOfficial/configService"
)

//ConfigStruct config for the server
type ConfigStruct struct {
	Server configServer

	Webserver struct {
		MaxHeaderLength uint  `default:"8000" required:"true"`
		MaxBodyLength   int64 `default:"10000" required:"true"`
		HTTP            configHTTPstruct
		HTTPS           configTLSStruct
	}
}

type configWhBlacklist struct {
	HeaderValues map[string][]string
	JSONObjects  map[string][]string
}

type configRetries struct {
	RetryTimes         map[uint8]time.Duration
	RetryInterval      time.Duration `required:"true"`
	InvalidUserRetries uint8         `required:"true" default:"2"`
}

type configServer struct {
	Database             configDBstruct
	WebhookBlacklist     configWhBlacklist
	AllowRegistration    bool `default:"false"`
	BogonAsCallback      bool `default:"false"`
	ServerHostAsCallback bool `default:"false"`
	WorkerCount          int  `default:"8"`
	Retries              configRetries
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
	Enabled       bool   `default:"false"`
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
				AllowRegistration:    false,
				BogonAsCallback:      false,
				ServerHostAsCallback: false,
				Retries: configRetries{
					RetryTimes: map[uint8]time.Duration{
						0: 1 * time.Minute,
						1: 10 * time.Minute,
						2: 30 * time.Minute,
						3: 60 * time.Minute,
						4: 2 * time.Hour,
						5: 10 * time.Hour,
					},
					RetryInterval:      10 * time.Second,
					InvalidUserRetries: 2,
				},
				Database: configDBstruct{
					Host:         "localhost",
					DatabasePort: 3306,
				},
				WebhookBlacklist: configWhBlacklist{
					HeaderValues: map[string][]string{
						"x-github-event": []string{
							"ping",
							"deploy_key",
						},
					},
					JSONObjects: map[string][]string{
						"github": []string{},
						"gitlab": []string{},
						"docker": []string{
							"callback_url",
						},
					},
				},
			},
			Webserver: struct {
				MaxHeaderLength uint  `default:"8000" required:"true"`
				MaxBodyLength   int64 `default:"10000" required:"true"`
				HTTP            configHTTPstruct
				HTTPS           configTLSStruct
			}{
				MaxHeaderLength: 8000,
				MaxBodyLength:   10000,
				HTTP: configHTTPstruct{
					Enabled:       true,
					ListenAddress: ":80",
				},
				HTTPS: configTLSStruct{
					Enabled:       false,
					ListenAddress: ":443",
				},
			},
		}
	}

	isDefault, err := configService.SetupConfig(&config, confFile, configService.NoChange)
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

	if err = configService.Load(&config, confFile); err != nil {
		log.Fatalln(err.Error())
		return nil, true
	}

	config.LoadInfo()

	return &config, false
}

//LoadInfo prints debugging config information
func (config *ConfigStruct) LoadInfo() {
	docker, gh, gl := 0, 0, 0
	jsonobjects := config.Server.WebhookBlacklist.JSONObjects

	d, has := jsonobjects["docker"]
	if has {
		docker = len(d)
	}
	d, has = jsonobjects["github"]
	if has {
		gh = len(d)
	}
	d, has = jsonobjects["gitlab"]
	if has {
		gl = len(d)
	}

	log.Infof("Blacklist: (%dx docker, %dx github, %dx gitlab) JSONObjects to block\n", docker, gh, gl)
	log.Infof("Blocking header values: %d\n", len(config.Server.WebhookBlacklist.HeaderValues))
}

//Check check the config file of logical errors
func (config *ConfigStruct) Check() bool {
	if !config.Webserver.HTTP.Enabled && !config.Webserver.HTTPS.Enabled {
		log.Error("You must at least enable one of the server protocols!")
		return false
	}

	if config.Webserver.HTTPS.Enabled {
		if len(config.Webserver.HTTPS.CertFile) == 0 || len(config.Webserver.HTTPS.KeyFile) == 0 {
			log.Error("If you enable TLS you need to set CertFile and KeyFile!")
			return false
		}
		//Check SSL files
		if !gaw.FileExists(config.Webserver.HTTPS.CertFile) {
			log.Error("Can't find the SSL certificate. File not found")
			return false
		}
		if !gaw.FileExists(config.Webserver.HTTPS.KeyFile) {
			log.Error("Can't find the SSL key. File not found")
			return false
		}
	}

	if config.Server.Database.DatabasePort < 1 || config.Server.Database.DatabasePort > 65535 {
		log.Errorf("Invalid port for database %d\n", config.Server.Database.DatabasePort)
		return false
	}

	return true
}
