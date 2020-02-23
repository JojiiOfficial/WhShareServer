package main

//StrToMode string to mode
var StrToMode = map[string]uint8{
	"custom": 0,
	"gitlab": 1,
	"docker": 2,
	"github": 3,
}

//ModeToString mode to string
var ModeToString = map[uint8]string{
	0: "custom",
	1: "gitlab",
	2: "docker",
	3: "github",
}

//Modes the available actions
var Modes = map[string]uint8{
	"github": 3, "gitlab": 1, "docker": 2, "script": 0,
}

const (
	//EPPingClient endpoint for pinging the client
	EPPingClient = "ping"
)

//LogLevels
var (
	LogLevels = []string{
		"debug",
		"info",
		"warning",
		"error",
	}
)
