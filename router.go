package main

import (
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
)

var templates = template.Must(template.ParseFiles("tmpl/view.html", "tmpl/edit.html",
	"tmpl/edit2.html", "tmpl/editcat.html", "tmpl/compare.html", "tmpl/multicompare.html",
	"tmpl/consolidate.html", "tmpl/tree.html", "tmpl/crud.html", "tmpl/login.html", "tmpl/callback.html",
	"tmpl/main.html"))

var jstemplates = template.Must(template.ParseFiles("js/ict2.js"))

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
	router.HandleFunc("/main/", MainPage)                         //So far this is just the page, a user is redirected to after login
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
