package main

import (
	"flag"
)

var noAuth *bool
var configLocation *string
var localAssets *bool
var checkForUpdates *bool

func initializeFlags() {
	noAuth = flag.Bool("noauth", false, "Start Brucheion without authenticating with a provider. (default: false)")
	configLocation = flag.String("config", "./config.json", "Specify where to load the JSON config from. (default: ./config.json")
	localAssets = flag.Bool("localAssets", false, "Obtain static assets from the local filesystem during development. (default: false)")
	checkForUpdates = flag.Bool("update", false, "Check for updates and install them at startup. (default: false)")

	flag.Parse()
}
