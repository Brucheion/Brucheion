package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

//The configuration that is needed for for the cookiestore. Holds Host information and provider secrets.
var config Config

var templates = template.Must(template.ParseFiles("tmpl/view.html", "tmpl/edit.html",
	"tmpl/edit2.html", "tmpl/editcat.html", "tmpl/compare.html", "tmpl/multicompare.html",
	"tmpl/consolidate.html", "tmpl/tree.html", "tmpl/crud.html", "tmpl/login.html", "tmpl/callback.html",
	"tmpl/main.html"))

var jstemplates = template.Must(template.ParseFiles("js/ict2.js"))

//Main starts the program the mux server
func main() {

	//evaluates flags and sets variables accordingly
	initializeFlags()

	if *configLocation != "./config.json" {
		log.Println("Loading configuration from: " + *configLocation)
		config = loadConfiguration(*configLocation)
	} else {
		log.Println("Loading configuration from: ./config.json")
		config = loadConfiguration("./config.json")
	}

	//Create new Cookiestore instance for use with Brucheion
	BrucheionStore = getCookieStore(config.MaxAge)

	if !*noAuth { //If Brucheion is NOT started in noAuth mode:
		//Set up gothic for authentification using the helper function
		setUpGothic()
	} else {
		log.Println("Started in noAuth mode.")
	}

	//Create new router instance with associated routes
	router := setUpRouter()

	log.Println("Listening at " + config.Host + "...")
	log.Fatal(http.ListenAndServe(config.Port, router))
}

//LandingPage is the first landing page for experimental testing
func LandingPage(res http.ResponseWriter, req *http.Request) {

	session, err := getSession(req)
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
		return
	}

	//log.Printf("func MainPage: User still known? Should be: %s\n", user)

	dbname := config.UserDB

	buckets := Buckets(dbname)
	log.Println()
	log.Printf("func MainPage. Printing buckets of %s:\n", dbname)
	log.Println()
	log.Println(buckets)

	//test := BoltRetrieve(dbname, "users", "test")
	adri := BoltRetrieve(dbname, "users", "adri")

	log.Println("User test:")
	log.Println(BoltRetrieve(dbname, "users", "test"))
	log.Println("User adri:")
	log.Println(adri)
	//log.Println("user adri: " + BoltRetrieve(dbname, users, adri) + "\n")

	page := &Page{
		User: user,
		Host: config.Host}
	renderTemplate(res, "main", page)
}

func setUpRouter() *mux.Router {

	//Start the router
	router := mux.NewRouter().StrictSlash(true)

	//Set up handlers for serving static files
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.Dir("./static/")))
	jsHandler := http.StripPrefix("/js/", http.FileServer(http.Dir("./js/")))
	cexHandler := http.StripPrefix("/cex/", http.FileServer(http.Dir("./cex/")))

	//Set up PathPrefix routes for serving static files
	router.PathPrefix("/static/").Handler(staticHandler)
	router.PathPrefix("/js/").Handler(jsHandler)
	router.PathPrefix("/cex/").Handler(cexHandler)

	//Set up HandleFunc routes
	router.HandleFunc("/login/", LoginGET).Methods("GET")         //The initial page. Idially the page users start from. This is where users are redirected to if not logged in corectly. Displays an error message.
	router.HandleFunc("/login/", LoginPOST).Methods("POST")       //This is where users are redirected to when credentials habe been entered.
	router.HandleFunc("/auth/{provider}/", Auth)                  //Initializes the authentication, redirects to callback.
	router.HandleFunc("/auth/{provider}/callback/", AuthCallback) //Displays message when logged in successfully. Forwards to Main
	router.HandleFunc("/logout/", Logout)                         //Logs out the User. Moved to helper.go
	router.HandleFunc("/{urn}/treenode.json/", Treenode)          //Function at treeBank.go
	router.HandleFunc("/main/", LandingPage)                      //So far this is just the page, a user is redirected to after login
	router.HandleFunc("/load/{cex}/", LoadCEX)
	router.HandleFunc("/new/{key}/{updated}", newText)
	router.HandleFunc("/view/{urn}/", ViewPage)
	router.HandleFunc("/tree/", TreePage)
	router.HandleFunc("/multicompare/{urn}/", MultiPage)
	router.HandleFunc("/edit/{urn}/", EditPage)
	router.HandleFunc("/editcat/{urn}/", EditCatPage)
	router.HandleFunc("/save/{key}/", SaveTranscription)
	router.HandleFunc("/addNodeAfter/{key}/", AddNodeAfter)
	router.HandleFunc("/addFirstNode/{key}/", AddFirstNode)
	router.HandleFunc("/crud/", CrudPage)
	router.HandleFunc("/deleteBucket/{urn}/", deleteBucket)
	router.HandleFunc("/deleteNode/{urn}/", deleteNode)
	router.HandleFunc("/export/{filename}/", ExportCEX)
	router.HandleFunc("/edit2/{urn}/", Edit2Page)
	router.HandleFunc("/compare/{urn}+{urn2}/", comparePage)
	router.HandleFunc("/consolidate/{urn}+{urn2}/", consolidatePage)
	router.HandleFunc("/saveImage/{key}/", SaveImageRef)
	router.HandleFunc("/newWork/", newWork)
	router.HandleFunc("/newCollection/{name}/{urns}/", newCollection)
	router.HandleFunc("/newCITECollection/{name}/", newCITECollection)
	router.HandleFunc("/getImageInfo/{name}/{imageurn}", getImageInfo)
	router.HandleFunc("/addtoCITE/", addCITE)
	router.HandleFunc("/requestImgID/{name}", requestImgID)
	router.HandleFunc("/deleteCollection", deleteCollection)
	router.HandleFunc("/requestImgCollection", requestImgCollection)
	router.NotFoundHandler = http.HandlerFunc(NotFoundRedirect)

	return router
}

//NotFoundRedirect redirects user to login in case an invalid request was issued.
func NotFoundRedirect(res http.ResponseWriter, req *http.Request) {
	newLink := config.Host + "/login/"
	http.Redirect(res, req, newLink, 301)
}
