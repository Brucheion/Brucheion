package main

import (
	"log"
	"net/http"
)

//The configuration that is needed for for the cookiestore. Holds Host information and provider secrets.
var config Config

//Main starts the program the mux server
func main() {

	//evaluates flags and sets variables accordingly
	initializeFlags()

	if *configLocation != "./config.json" {
		log.Println("loading configuration from: " + *configLocation)
		config = LoadConfiguration(*configLocation)
	} else {
		config = LoadConfiguration("./config.json")
	}

	//Create new Cookiestore instance for use with Brucheion
	BrucheionStore = GetCookieStore(config.MaxAge)

	if !*noAuth { //If Brucheion is NOT started in noAuth mode:
		//Set up gothic for authentification using the helper function
		SetUpGothic()
	}

	//Create new router instance with associated routes
	router := setUpRouter()

	log.Println("Listening at " + config.Host + "...")
	log.Fatal(http.ListenAndServe(config.Port, router))
}

//First landing page for experimental testing
func MainPage(res http.ResponseWriter, req *http.Request) {

	session, err := GetSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	user, message, loggedin := TestLoginStatus("MainPage", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
	}

	//log.Printf("func MainPage: User still known? Should be: %s\n", user)

	dbname := config.UserDB

	buckets := Buckets(dbname)
	log.Println()
	log.Printf("func MainPage. Printing buckets of %s:\n", dbname)
	log.Println()
	log.Println(buckets)

	page := &Page{
		User: user,
		Host: config.Host}
	renderTemplate(res, "main", page)
}
