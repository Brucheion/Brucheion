package main

import (
	"flag"
)

var noAuth *bool
var configLocation *string

// initializeFlags defines how the flags are to behave
//noAuth switches the login behavior. If set true, authentication via a safe login provider is circumvent and only a user name is needed for login.
//configLocation can be used to specify the location of the config file when needed.
func initializeFlags() {
	noAuth = flag.Bool("noauth", false, "Start Brucheion without authentificating with a provider (default: false)")

	configLocation = flag.String("config", "./config.json", "Specify where to load the JSON config from. (defalult: ./config.json")

	flag.Parse()
}
