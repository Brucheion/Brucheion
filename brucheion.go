package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/skratchdot/open-golang/open"
)

var config Config
var providers Providers
var templates *template.Template

var BuildTime = ""
var Version = "development"

var dataPath string
var err error

//go:embed tmpl static js ui/dist
var assets embed.FS

func main() {
	initializeFlags()

	if Version == "development" {
		fmt.Println("This is a development build of Brucheion.")
	} else {
		fmt.Printf("Brucheion %s, built %s\n", Version, BuildTime)
	}

	if *checkForUpdates {
		if Version == "development" {
			log.Println("Development builds can't self-update.")
		} else if handleUpdates() {
			os.Exit(0)
		}
	}

	if *localAssets && Version == "development" {
		log.Println("Serving static assets from the local filesystem.")
	} else if *localAssets && Version != "development" {
		log.Println("Production builds can't serve static assets from the local filesystem.")
	}

	dataPath, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Data path: %s\n", dataPath)

	providers, err = loadProviders()
	if err != nil {
		log.Fatalf("Loading authentication providers failed: %s\n", err.Error())
	}

	cp := filepath.Join(dataPath, "config.json")
	if *configLocation != "" {
		cp = *configLocation
	}
	log.Printf("Loading configuration from: %s\n", cp)
	config, err = loadConfiguration(cp)
	if err != nil {
		log.Fatalf("Loading configuration failed: %s\n", err.Error())
	}

	t := createBaseTemplate()
	templates, err = t.ParseFS(mustFS(fs.Sub(assets, "tmpl")), "*.html", "shared/*.html")
	if err != nil {
		log.Fatal(err)
	}

	//Create new Cookiestore instance for use with Brucheion
	BrucheionStore = getCookieStore(config.MaxAge)

	if !*noAuth { //If Brucheion is NOT started in noAuth mode:
		//Set up gothic for authentification using the helper function
		setUpGothic()
	} else {
		log.Println("Started in noAuth mode.")
	}

	router := createRouter()

	log.Printf("Listening at %s\n", config.Host)
	l, err := net.Listen("tcp", config.Port)
	if err != nil {
		log.Fatal(err)
	}

	if Version != "development" {
		err = open.Start(config.Host)
		if err != nil {
			log.Println(err)
		}
	}

	log.Fatal(http.Serve(l, router))
}

//landingPage is the first landing page for experimental testing
func landingPage(res http.ResponseWriter, req *http.Request) {
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	user, message, loggedin := testLoginStatus("MainPage", session)
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
	adri, _ := BoltRetrieve(dbname, "users", "adri")

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

// mustFS is a helper that wraps a call to a function returning (*fs.FS, error)
// and panics if the error is non-nil.
func mustFS(f fs.FS, err error) fs.FS {
	if err != nil {
		log.Fatalln(err)
	}
	return f
}

func createRouter() *mux.Router {
	//Start the router
	router := mux.NewRouter().StrictSlash(true)
	a := router.PathPrefix("/api/v1").Subrouter()

	staticDir := http.FS(mustFS(fs.Sub(assets, "static")))
	jsDir := http.FS(mustFS(fs.Sub(assets, "js")))
	bundleDir := http.FS(mustFS(fs.Sub(assets, "ui/dist")))

	if *localAssets {
		staticDir = http.Dir("./static")
		jsDir = http.Dir("./js")
		bundleDir = http.Dir("./ui/dist")
	}

	//Set up handlers for serving static files
	libraryHandler := http.StripPrefix("/static/image_archive", http.FileServer(http.Dir("./image_archive")))
	staticHandler := http.StripPrefix("/static/", http.FileServer(staticDir))
	jsHandler := http.StripPrefix("/js/", http.FileServer(jsDir))
	cexHandler := http.StripPrefix("/cex/", http.FileServer(http.Dir("./cex/")))
	bundleHandler := http.StripPrefix("/assets/ui", http.FileServer(bundleDir))

	//Set up PathPrefix routes for serving static files
	router.PathPrefix("/static/image_archive/").Handler(libraryHandler)
	router.PathPrefix("/static/").Handler(staticHandler)
	router.PathPrefix("/js/").Handler(jsHandler)
	router.PathPrefix("/cex/").Handler(cexHandler)
	router.PathPrefix("/assets/ui").Handler(bundleHandler)

	//Set up HandleFunc routes
	router.HandleFunc("/login/", loginGET).Methods("GET")         //The initial page. Ideally the page users start from. This is where users are redirected to if not logged in corectly. Displays an error message.
	router.HandleFunc("/login/", loginPOST).Methods("POST")       //This is where users are redirected to when credentials habe been entered.
	router.HandleFunc("/auth/{provider}/", auth)                  //Initializes the authentication, redirects to callback.
	router.HandleFunc("/auth/{provider}/callback/", authCallback) //Displays message when logged in successfully. Forwards to Main
	router.HandleFunc("/logout/", Logout)                         //Logs out the User. Moved to helper.go
	router.HandleFunc("/{urn}/treenode.json/", Treenode)          //Function at treeBank.go
	router.HandleFunc("/main/", landingPage)                      //So far this is just the page, a user is redirected to after login
	router.HandleFunc("/new/{key}/{updated}/", newText)
	router.HandleFunc("/view/{urn}/", ViewPage)
	router.HandleFunc("/tree/", TreePage)
	router.HandleFunc("/ingest/cex", createSpaHandler("CEX Ingestion"))
	router.HandleFunc("/ingest/image", createSpaHandler("Image Ingestion"))
	router.HandleFunc("/multicompare/{urn}/", MultiPage).Methods("GET")
	router.HandleFunc("/seealignment/{urn}", SeeAlignment).Methods("GET")
	router.HandleFunc("/tablealignment/{urn}", TableAlignments).Methods("GET")
	router.HandleFunc("/normalizeTemporarily/{urns}/", normalizeOrthographyTemporarily)
	router.HandleFunc("/normalizeAndSave/{urns}/", normalizeOrthographyAndSave)
	router.HandleFunc("/edit/{urn}/", EditPage)
	router.HandleFunc("/edit/{urn}/{format}", EditPageFormat)
	// router.Path("/edit/{urn}").Queries("format", "{[a-Z]+}").HandlerFunc(EditPageFormat).Name("EditPageFormat")
	router.HandleFunc("/editcat/{urn}/", EditCatPage)
	router.HandleFunc("/save/{key}/", SaveTranscription)
	//	router.HandleFunc("/addNodeAfter/{key}/", AddNodeAfter)
	router.HandleFunc("/addFirstNode/{key}/", AddFirstNode)
	router.HandleFunc("/crud/", CrudPage)
	router.HandleFunc("/deleteBucket/{urn}/", deleteBucket)
	router.HandleFunc("/deleteNode/{urn}/", deleteNode)
	router.HandleFunc("/export/{filename}/", ExportCEX)
	router.HandleFunc("/edit2/{urn}/", Edit2Page)
	router.HandleFunc("/compare/{urn}+{urn2}/", comparePage)
	router.HandleFunc("/consolidate/{urn}+{urn2}/", consolidatePage)
	router.HandleFunc("/saveImage/{key}/{updated}/", SaveImageRef)
	router.HandleFunc("/newWork/", newWork)
	router.HandleFunc("/newCollection/{name}/{urns}/", newCollection)
	router.HandleFunc("/newCITECollection/{name}/", newCITECollection)
	router.HandleFunc("/getImageInfo/{name}/{imageurn}", getImageInfo)
	router.HandleFunc("/addtoCITE/", addCITE)
	router.HandleFunc("/requestImgID/{name}/", requestImgID)
	router.HandleFunc("/deleteCollection/", deleteCollection)
	router.HandleFunc("/requestImgCollection/", requestImgCollection)
	router.HandleFunc("/favicon.ico", FaviconHandler)

	// API routes
	a.HandleFunc("/cex/upload", requireSession(handleCEXUpload))

	// legacy redirects
	router.HandleFunc("/ingest", createPermanentRedirect("/ingest/image"))

	router.NotFoundHandler = http.HandlerFunc(NotFoundRedirect)

	return router
}

func createPermanentRedirect(path string) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		http.Redirect(res, req, config.Host+path, http.StatusMovedPermanently)
	}
}

//NotFoundRedirect redirects user to login in case an invalid request was issued.
func NotFoundRedirect(res http.ResponseWriter, req *http.Request) {
	newLink := config.Host + "/login/"
	http.Redirect(res, req, newLink, 301)
}

//FaviconHandler returns the favicon to browsers
func FaviconHandler(res http.ResponseWriter, req *http.Request) {
	log.Println("FaviconHandler reporting.")
	http.ServeFile(res, req, "static/img/favicon.png")
}
