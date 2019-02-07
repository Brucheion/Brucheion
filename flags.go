package main

import (
	"flag"
)

var noAuth *bool
var configLocation *string

func initializeFlags() {
	noAuth = flag.Bool("noauth", false, "Start Brucheion without authentificating with a provider (default: false)")

	configLocation = flag.String("config", "./config.json", "Specify where to load the JSON config from. (defalult: ./config.json")

	flag.Parse()
}
