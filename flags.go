package main

import (
	"flag"
)

var noAuth *bool
var configLocation *string
var localAssets *bool
var checkForUpdates *bool
var heroku *bool

func initializeFlags() {
	noAuth = flag.Bool("noauth", false, "Start Brucheion without authenticating with a provider. (default: false)")
	configLocation = flag.String("config", "", "Specify where to load the JSON config from. (default: from data directory)")
	localAssets = flag.Bool("localAssets", false, "Obtain static assets from the local filesystem during development. (default: false)")
	checkForUpdates = flag.Bool("update", false, "Check for updates and install them at startup. (default: false)")
	heroku = flag.Bool("heroku", false, "Deploying on heroku. (default: false)")

	flag.Parse()
}
