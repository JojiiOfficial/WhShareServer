package main

import (
	"fmt"
)

//Env vars
const (
	//EnVarPrefix prefix of all used env vars
	EnVarPrefix = "S"

	EnVarNoColor    = "NOCOLOR"
	EnVarYes        = "SKIP_CONFIRM"
	EnVarConfigFile = "CONFIG"
)

func getEnVar(name string) string {
	return fmt.Sprintf("%s_%s", EnVarPrefix, name)
}
