package main

const (
	//DefaultConfigFile default config filename
	DefaultConfigFile = "config.yml"
	//DataDir the dir where the config and data is
	DataDir = "./data/"
)

//Modes the available actions
var Modes = map[string]uint8{
	"github": 3, "gitlab": 1, "docker": 2, "script": 0,
}
