package main

import (
	"fmt"
)

//Env vars
const (
	//EnVarPrefix prefix of all used env vars
	EnVarPrefix = "S"

	EnVarDebug      = "DEBUG"
	EnVarNoColor    = "NO-COLOR"
	EnVarYes        = "SKIP_CONFIRM"
	EnVarConfigFile = "CONFIG"
)

//Return the variable using the server prefix
func getEnVar(name string) string {
	return fmt.Sprintf("%s_%s", EnVarPrefix, name)
}
