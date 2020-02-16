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

const (
	//HeaderSubsID subscription for ping
	HeaderSubsID = "W_S_SubsID"
	//HeaderSource the sourceID of the incoming hook
	HeaderSource = "W_S_Source"
	//HeaderReceived the unix time when the hook was received
	HeaderReceived = "W_S_Source"
)

const (
	//EPPingClient endpoint for pinging the client
	EPPingClient = "ping"
)
