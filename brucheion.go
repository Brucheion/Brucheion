package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ThomasK81/gocite"
	"github.com/ThomasK81/gonwr"
	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"

	"github.com/gorilla/sessions" //for Cookiestore and other session functionality

	"github.com/markbates/goth/gothic"
)

//The Config stores Host/Port Information, where the user DB is and settings for the cookiestores
//
//Host and Port are used throughout brucheion for parsing and delivering the pages
//
//The Key/Secret pairs are obtained from the provider when registering the application.
type Config struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	GitHubKey    string `json:"gitHubKey"`
	GitHubSecret string `json:"githHubSecret"`
	GitLabKey    string `json:"gitLabKey"`
	GitLabSecret string `json:"gitLabSecret"`
	GitLabScope  string `json:"gitLabScope"` //for accessing GitLab user information this has to be "read_user"
	MaxAge       int    `json:"maxAge"`      //defines the lifetime of the brucheion session
	UserDB       string `json:"userDB"`
	//	GoogleKey	  string `json:"googleKey"`
	//	GoogleSecret  string `json:"googleSecret"`
}

type JSONlist struct {
	Item []string `json:"item"`
}

type Transcription struct {
	CTSURN        string
	Transcriber   string
	Transcription string
	Previous      string
	Next          string
	First         string
	Last          string
	ImageRef      []string
	TextRef       []string
	ImageJS       string
	CatID         string
	CatCit        string
	CatGroup      string
	CatWork       string
	CatVers       string
	CatExmpl      string
	CatOn         string
	CatLan        string
}

type CompPage struct {
	User      string
	Title     string
	Text      template.HTML
	Host      string
	CatID     string
	CatCit    string
	CatGroup  string
	CatWork   string
	CatVers   string
	CatExmpl  string
	CatOn     string
	CatLan    string
	User2     string
	Title2    string
	Text2     template.HTML
	CatID2    string
	CatCit2   string
	CatGroup2 string
	CatWork2  string
	CatVers2  string
	CatExmpl2 string
	CatOn2    string
	CatLan2   string
}

type Page struct {
	User         string
	Title        string
	ImageJS      string
	ImageScript  template.HTML
	ImageHTML    template.HTML
	TextHTML     template.HTML
	InTextHTML   template.HTML
	Text         template.HTML
	Previous     string
	Next         string
	PreviousLink template.HTML
	NextLink     template.HTML
	First        string
	Last         string
	Host         string
	ImageRef     string
	CatID        string
	CatCit       string
	CatGroup     string
	CatWork      string
	CatVers      string
	CatExmpl     string
	CatOn        string
	CatLan       string
}

//LoginPage stores Information necessary to parse and display /login/ and /auth/{provider}/callback pages
type LoginPage struct {
	BUserName    string //The username that the user chooses to work with within Brucheion
	Provider     string //The login provider
	HrefUserName string //Combination {user}_{provider} as displayed in link
	Message      string //Message to be displayed according to login scenario
	Host         string //Port of the Link
	Title        string //Title of the website
}

//BrucheionUser stores Information about the logged in Brucheion-user
type BrucheionUser struct {
	BUserName      string //The username choosen by user to use Brucheion with
	Provider       string //The provider used for authentification
	PUserName      string //The username used for login with the provider
	ProviderUserID string //The UserID issued by Provider
}

//Validation stores the result of the validation
type Validation struct {
	Message      string //Message according to outcome of validation
	ErrorCode    bool   //Was an error encountered during validation (something did not match)?
	BUserInUse   bool   //func ValidateUser: Is the BrucheionUser to be found in the DB?
	SameProvider bool   //func ValidateUser: Is the chosen provider the same as the providersaved in DB?
	PUserInUse   bool   //func ValidateUser: Is the ProviderUser to be found in the DB?
}

//The configuration that is needed for for the cookiestore. Holds Host information and provider secrets.
var config = LoadConfiguration("./config.json")

var templates = template.Must(template.ParseFiles("tmpl/view.html", "tmpl/edit.html",
	"tmpl/edit2.html", "tmpl/editcat.html", "tmpl/compare.html", "tmpl/multicompare.html",
	"tmpl/consolidate.html", "tmpl/tree.html", "tmpl/crud.html", "tmpl/login.html", "tmpl/callback.html",
	"tmpl/main.html"))
var jstemplates = template.Must(template.ParseFiles("js/ict2.js"))

//The sessionName of the Brucheion Session
const SessionName = "brucheionSession"

//
var BrucheionStore sessions.Store

//Main initializes the mux server
func main() {

	//
	SetUpGothic()

	//Create new Cookiestore for Brucheion
	BrucheionStore = GetCookieStore(config.MaxAge)

	//Set up the router
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
	router.HandleFunc("/load/{cex}/", LoadDB)
	router.HandleFunc("/new/{key}/", newText)
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
	router.HandleFunc("/getImageInfo/{name}/{imageurn}/", getImageInfo)
	router.HandleFunc("/addtoCITE/", addCITE)
	router.HandleFunc("/requestImgID/{name}", requestImgID)
	router.HandleFunc("/deleteCollection", deleteCollection)
	router.HandleFunc("/requestImgCollection", requestImgCollection)
	log.Println("Listening at " + config.Host + "...")
	log.Fatal(http.ListenAndServe(config.Port, router))
}

//LoginGET renders the login page. The user can enter the login Credentials into the form.
//If already logged in, the user will be redirected to main page.
func LoginGET(res http.ResponseWriter, req *http.Request) {

	//Make sure user is not logged in yet
	session, err := GetSession(req) //Get a session
	if err != nil {
		fmt.Errorf("LoginGET: Error getting session: %s", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if session.Values["Loggedin"] != nil { //test if the Loggedin variable has already been set
		if session.Values["Loggedin"].(bool) { //"Loggedin" will be true if user is already logged in
			user, ok := session.Values["BrucheionUserName"].(string) //if session was valid get a username
			if !ok {
				fmt.Println("func LoginGET: Type assertion to string failed for session value BrucheionUser or session value could not be retrieved.")
			}
			fmt.Printf("User %s is already logged in. Redirecting to main", user) //use the username for debugging
			http.Redirect(res, req, "/main/", http.StatusFound)
		}
	} else { //Destroy the newly created session if Loggedin was not set
		session.Options.MaxAge = -1
		session.Values = make(map[interface{}]interface{})
		err = session.Save(req, res)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	lp := &LoginPage{
		Title: "Brucheion Login Page"}
	renderLoginTemplate(res, "login", lp)
}

//LoginPOST logs in the user using the form values and gothic.
func LoginPOST(res http.ResponseWriter, req *http.Request) {
	//Make sure user is not logged in yet
	session, err := GetSession(req) //get a session
	if err != nil {
		fmt.Errorf("LoginGET: Error getting session: %s", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if session.Values["Loggedin"] != nil { //If the Loggedin variable has been set already..
		if session.Values["Loggedin"].(bool) { //And if "Loggedin" is true..
			user, ok := session.Values["BrucheionUserName"].(string) //Then get the username
			if !ok {
				fmt.Println("func LoginPOST: Type assertion to string failed for session value BrucheionUser or session value could not be retrieved.")
			}
			fmt.Printf("User %s is already logged in. Redirecting to main\n", user) //Log that session was already logged in
			http.Redirect(res, req, "/main/", http.StatusFound)                     //redirect to main, as login is not necessary anymore
		}
	} else { //Destroy the session we just got if was not logged in yet (proceed with login process)
		session.Options.MaxAge = -1
		session.Values = make(map[interface{}]interface{})
		err = session.Save(req, res)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	title := "Brucheion Login Page"

	/*fmt.Println("req.FormValue(\"brucheionUserName\")" + req.FormValue("brucheionusername"))
	fmt.Println("req.FormValue(\"provider\")" + req.FormValue("provider"))*/

	//populates Loginpage with basic data and the form values
	lp := &LoginPage{
		BUserName: strings.TrimSpace(req.FormValue("brucheionusername")),
		Provider:  req.FormValue("provider"),
		Title:     title}

	unameValidation := ValidateUserName(lp.BUserName) //checks if this username only has (latin) letters and (arabian) numbers

	authPath := "/auth/" + strings.ToLower(lp.Provider) + "/" //set up the path for redirect according to provider

	/*fmt.Println("authPath: " + authPath)
	fmt.Println("userDB:" + config.UserDB)*/

	if unameValidation.ErrorCode { //if a valid username has been chosen
		session, err = InitializeSession(req) //initialize a persisting session
		if err != nil {
			fmt.Println("LoginPOST: Error initializing the session.")
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		//save the values from the form in the session
		session.Values["BrucheionUserName"] = lp.BUserName
		session.Values["Provider"] = lp.Provider            //the provider used for login
		session.Values["Loggedin"] = false                  //make sure the "Loggedin" session value is set but false
		session.Save(req, res)                              //always save the session after setting values
		http.Redirect(res, req, authPath, http.StatusFound) //redirect to auth page with correct provider
	} else { //if the the user name was not valid
		lp.Message = unameValidation.Message  //add the message to the loginpage..
		renderLoginTemplate(res, "login", lp) //and render the login template again, displaying said message.
	}
}

//Auth redirects to provider for authentification using gothic.
//Provider redirects to callback page.
func Auth(res http.ResponseWriter, req *http.Request) {
	//Make sure user is not logged in yet
	session, err := GetSession(req) //get a session
	if err != nil {
		fmt.Errorf("LoginGET: Error getting session: %s", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if session.Values["Loggedin"] != nil { //If the Loggedin variable has been set already
		if session.Values["Loggedin"].(bool) { //And if "Loggedin" is true
			user, ok := session.Values["BrucheionUserName"].(string) //Then get the username
			if !ok {
				fmt.Println("func Auth: Type assertion to string failed for session value BrucheionUser or session value could not be retrieved.")
			}
			fmt.Printf("User %s is already logged in. Redirecting to main\n", user) //Log that session was already logged in
			http.Redirect(res, req, "/main/", http.StatusFound)                     //redirect to main, as login is not necessary anymore
		} else { //proceed with login process
			gothic.BeginAuthHandler(res, req)
		}
	} else { //kill the session and redirect to login
		fmt.Println("func Auth: \"Loggedin\" was nil. Session was not initialized. Logging out")
		Logout(res, req)
	}
}

//AuthCallback complets user authentification, sets session variables and DB entries.
func AuthCallback(res http.ResponseWriter, req *http.Request) {
	//Make sure user is not logged in yet
	session, err := GetSession(req) //get a session
	if err != nil {
		fmt.Errorf("LoginGET: Error getting session: %s", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if session.Values["Loggedin"] != nil { //If the Loggedin variable has been set already
		if session.Values["Loggedin"].(bool) { //And if "Loggedin" is true
			user, ok := session.Values["BrucheionUserName"].(string) //Then get the username
			if !ok {
				fmt.Println("func AuthCallback: Type assertion to string failed for session value BrucheionUser or session value could not be retrieved.")
			}
			fmt.Printf("User %s is already logged in. Redirecting to main\n", user) //Log that session was already logged in
			http.Redirect(res, req, "/main/", http.StatusFound)                     //redirect to main, as login is not necessary anymore
		} //else proceed with login process
	} else { //kill the session and redirect to login
		fmt.Println("func AuthCallback: \"Loggedin\" was nil. Session was not initialized.")
		Logout(res, req)
	}

	err = InitializeUserDB() //Make sure the userDB file is there and has the necessary buckets.
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	gothUser, err := gothic.CompleteUserAuth(res, req) //authentificate user and get gothUser from gothic
	if err != nil {
		fmt.Fprintln(res, err)
		return
	}

	//get provider and username from session values
	provider, ok := session.Values["Provider"].(string)
	if !ok {
		fmt.Println("Func AuthCallback: Type assertion of value Provider to string failed or session value could not be retrieved.")
	}
	brucheionUserName, ok := session.Values["BrucheionUserName"].(string)
	if !ok {
		fmt.Println("Func AuthCallback: Type assertion of value BrucheionUserName to string failed or session value could not be retrieved.")
	}

	//save values retrieved from gothUser in session
	session.Values["Loggedin"] = false                     //assumed for later use, maybe going to be deprecated later
	session.Values["ProviderNickName"] = gothUser.NickName //The nickname used for logging in with provider
	session.Values["ProviderUserID"] = gothUser.UserID     //the userID returned by the login from provider
	session.Save(req, res)                                 //always remember to save the session

	validation, err := ValidateUser(req) //validate if credentials match existing user
	if err != nil {
		fmt.Printf("\nAuthCallback error validating user: %s", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

	brucheionUser := &BrucheionUser{ //create Brucheionuser instance
		BUserName:      brucheionUserName,
		Provider:       provider,
		PUserName:      gothUser.NickName,
		ProviderUserID: gothUser.UserID}

	//Save user in DB and/or login user if user is valid. Redirect back to login page is not
	if validation.ErrorCode { //Login scenarios (1), (5)
		if validation.BUserInUse && validation.SameProvider && validation.PUserInUse { //Login scenario (1)
			session.Values["Loggedin"] = true
			session.Save(req, res)
			fmt.Println(validation.Message) //Display validation.Message if all went well.
		} else if !validation.BUserInUse && !validation.SameProvider && !validation.PUserInUse { //Login scenario (5)
			//create new enty for new BUser
			db, err := OpenBoltDB(config.UserDB) //open the userDB
			if err != nil {
				fmt.Printf("Error opening userDB: %s", err)
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			db.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket([]byte("users"))
				buffer, err := json.Marshal(brucheionUser) //Marshal user data
				if err != nil {
					fmt.Errorf("Failed marshalling user data for user %s: %s\n", brucheionUserName, err)
					return err
				}
				err = bucket.Put([]byte(brucheionUserName), buffer) //put user into bucket
				if err != nil {
					fmt.Errorf("Failed saving user %s in users.db\n", brucheionUserName, err)
					return err
				}
				fmt.Printf("Successfully saved new user %s in users.DB.\n", brucheionUserName)

				bucket = tx.Bucket([]byte(provider))
				err = bucket.Put([]byte(brucheionUser.ProviderUserID), []byte(brucheionUserName))
				if err != nil {
					fmt.Errorf("Failed saving user ProviderUserID for user %s in Bucket %s.\n", brucheionUserName, provider, err)
					return err
				}
				fmt.Printf("Successfully saved ProviderUserID of BUser %s in Bucket %s.\n", brucheionUserName, provider)
				fmt.Println(validation.Message) //Display validation.Message if all went well.
				return nil
			})
			db.Close() //always remember to close the db
			//fmt.Println("DB closed")
			session.Values["Loggedin"] = true //To keep the user logged in
			session.Save(req, res)

		} else { //unknown login behavior
			fmt.Errorf("Unknown login behavior. This should never happen")
			//return errors.New("Unknown login behavior. This should never happen")
			return
		}
	} else { //Login scenarios (2), (3), (4)

		if (validation.BUserInUse && !validation.SameProvider && validation.PUserInUse) ||
			(!validation.BUserInUse && validation.SameProvider && validation.PUserInUse) ||
			(!validation.BUserInUse && validation.SameProvider && !validation.PUserInUse) { //unknown login behavior
			fmt.Errorf("Unknown login behavior. This should never happen")
			//return errors.New("Unknown login behavior. This should never happen")
			return
		} else {
			fmt.Println(validation.Message)
			validation.Message = validation.Message + "\nPlease always use the same combination of username, provider, and provider account."
			fmt.Println("Please always use the same combination of username, provider, and provider account.")
			lp := &LoginPage{
				Message: validation.Message}
			renderLoginTemplate(res, "login", lp)
			return
		}
	}

	lp := &LoginPage{
		Host:         config.Host,
		BUserName:    brucheionUserName,
		Provider:     provider,
		HrefUserName: brucheionUserName + "_" + provider,
		Message:      validation.Message} //The message to be replied in regard to the login scenario

	renderAuthTemplate(res, "callback", lp)

}

func MainPage(res http.ResponseWriter, req *http.Request) {

	session, err := GetSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	user, message, loggedin := TestLoginStatus("MainPage", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(res, req)
	}

	fmt.Printf("User still known? Should be: %s\n", user)

	dbname := config.UserDB
	/*
		db, err := bolt.Open(dbname, 0644, nil)
		if err != nil {
			fmt.Println("Error opening DB.")
			log.Fatal(err)
		}
		defer db.Close()*/

	buckets := Buckets(dbname)
	fmt.Println()
	fmt.Printf("func MainPage. Printing buckets of %s:\n", dbname)
	fmt.Println()
	fmt.Println(buckets)

	page := &Page{
		User: user,
		Host: config.Host}
	renderTemplate(res, "main", page)
}

func requestImgCollection(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("requestImgCollection", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	response := JSONlist{}
	dbname := user + ".db"
	db, err := OpenBoltDB(dbname)
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("imgCollection"))
		if b == nil {
			return errors.New("failed to get bucket")
		}
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			response.Item = append(response.Item, string(k))
		}
		return nil
	})
	if err != nil {
		resultJSON, _ := json.Marshal(response)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintln(w, string(resultJSON))
	}
	resultJSON, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(resultJSON))
}

func getImageInfo(w http.ResponseWriter, r *http.Request) {
	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("getImageInfo", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	retImage := imageCollection{}
	newImage := image{}
	vars := mux.Vars(r)
	collectionName := vars["name"]
	imageurn := vars["imageurn"]
	dbkey := []byte(collectionName)
	dbname := user + ".db"
	db, err := OpenBoltDB(dbname)
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("imgCollection"))
		if b == nil {
			return errors.New("failed to get bucket")
		}
		val := b.Get(dbkey)
		// fmt.Println("got", string(dbkey))
		retImage, _ = gobDecodeImgCol(val)
		for _, v := range retImage.Collection {
			if v.URN == imageurn {
				newImage = v
			}
		}
		return nil
	})
	if err != nil {
		resultJSON, _ := json.Marshal(newImage)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintln(w, string(resultJSON))
	}
	resultJSON, _ := json.Marshal(newImage)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(resultJSON))
}

func requestImgID(w http.ResponseWriter, r *http.Request) {
	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("requestImgID", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	response := JSONlist{}
	collection := imageCollection{}
	vars := mux.Vars(r)
	name := vars["name"]
	dbname := user + ".db"
	dbkey := []byte(name)
	db, err := OpenBoltDB(dbname)
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("imgCollection"))
		if bucket == nil {
			return errors.New("failed to get bucket")
		}
		val := bucket.Get(dbkey)
		if val == nil {
			return errors.New("failed to retrieve value")
		}
		collection, err = gobDecodeImgCol(val)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		resultJSON, _ := json.Marshal(response)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintln(w, string(resultJSON))
	}
	for i := range collection.Collection {
		response.Item = append(response.Item, collection.Collection[i].URN)
	}
	resultJSON, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(resultJSON))
}

func newCITECollection(w http.ResponseWriter, r *http.Request) {
	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("newCITECollection", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	vars := mux.Vars(r)
	name := vars["name"] //the name of the new CITE collection
	newCITECollectionDB(user, name)
	io.WriteString(w, "success")
}

func addCITE(w http.ResponseWriter, r *http.Request) {
	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("addCITE", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	// /thomas/addtoCITE?name="test"&urn="test"&internal="false"&protocol="static&location="https://digi.vatlib.it/iiifimage/MSS_Barb.lat.4/Barb.lat.4_0015.jp2/full/full/0/native.jpg"
	name := r.URL.Query().Get("name")
	name = strings.Replace(name, "\"", "", -1)
	imageurn := r.URL.Query().Get("urn")
	imageurn = strings.Replace(imageurn, "\"", "", -1)
	location := r.URL.Query().Get("location")
	location = strings.Replace(location, "\"", "", -1)
	// fmt.Println(location)
	protocol := r.URL.Query().Get("protocol")
	protocol = strings.Replace(protocol, "\"", "", -1)
	externalstr := r.URL.Query().Get("external")
	externalstr = strings.Replace(externalstr, "\"", "", -1)
	external := false
	if externalstr == "true" {
		external = true
	}
	newimage := image{URN: imageurn, External: external, Protocol: protocol, Location: location}
	// fmt.Println(user, name, newimage)
	addtoCITECollection(user, name, newimage)
	io.WriteString(w, "success")
}

func newCollection(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("newCollection", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	vars := mux.Vars(r)
	name := vars["name"]
	imageIDs := strings.Split(vars["urns"], ",")
	var collection imageCollection
	switch len(imageIDs) {
	case 0:
		io.WriteString(w, "failed")
		return
	case 1:
		urn := gocite.SplitCITE(imageIDs[0])
		switch {
		case urn.InValid:
			io.WriteString(w, "failed")
			return
		case urn.Object == "*":
			links, err := extractLinks(urn)
			if err != nil {
				io.WriteString(w, "failed")
			}
			for i := range links {
				collection.Collection = append(collection.Collection, image{External: false, Location: links[i]})
			}
		default:
			collection.Collection = append(collection.Collection, image{External: false, Location: imageIDs[0]})
		}
	default:
		for i := range imageIDs {
			urn := gocite.SplitCITE(imageIDs[i])
			switch {
			case urn.InValid:
				continue
			default:
				collection.Collection = append(collection.Collection, image{External: false, Location: imageIDs[i]})
			}
		}
	}
	newCollectiontoDB(user, name, collection)
	io.WriteString(w, "success")
}

func newWork(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("newWork", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	if r.Method == "GET" {
		varmap := map[string]interface{}{
			"user": user,
			"port": config.Port,
		}
		t, _ := template.ParseFiles("tmpl/newWork.html")
		t.Execute(w, varmap)
	} else {
		r.ParseForm()
		// logic part of log in
		workurn := r.Form["workurn"][0]
		scheme := r.Form["scheme"][0]
		group := r.Form["workgroup"][0]
		title := r.Form["title"][0]
		version := r.Form["version"][0]
		exemplar := r.Form["exemplar"][0]
		online := r.Form["online"][0]
		language := r.Form["language"][0]
		newWork := cexMeta{URN: workurn, CitationScheme: scheme, GroupName: group, WorkTitle: title, VersionLabel: version, ExemplarLabel: exemplar, Online: online, Language: language}
		fmt.Println(newWork)
		err := newWorktoDB(user, newWork)
		if err != nil {
			io.WriteString(w, "failed")
		} else {
			io.WriteString(w, "Success")
		}
	}
}

//TreePage loads and renders the Morpho-syntactic Treebank page
func TreePage(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("TreePage", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	dbname := user + ".db"

	textref := Buckets(dbname)

	transcription := Transcription{
		Transcriber: user,
		TextRef:     textref}
	p, _ := loadCrudPage(transcription)
	renderTemplate(w, "tree", p)
}

func CrudPage(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("CrudPage", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	dbname := user + ".db"

	textref := Buckets(dbname)

	transcription := Transcription{
		Transcriber: user,
		TextRef:     textref}
	p, _ := loadCrudPage(transcription)
	renderTemplate(w, "crud", p)
}

func ExportCEX(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("ExportCEX", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	var texturns, texts, areas, imageurns []string
	var indexs []int
	vars := mux.Vars(r)
	filename := vars["filename"]
	dbname := user + ".db"
	buckets := Buckets(dbname)
	db, err := OpenBoltDB(dbname)
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	for i := range buckets {
		db.View(func(tx *bolt.Tx) error {
			// Assume bucket exists and has keys
			b := tx.Bucket([]byte(buckets[i]))

			c := b.Cursor()

			for k, v := c.First(); k != nil; k, v = c.Next() {
				retrievedjson := BoltURN{}
				json.Unmarshal([]byte(v), &retrievedjson)
				ctsurn := retrievedjson.URN
				text := retrievedjson.Text
				index := retrievedjson.Index
				imageref := retrievedjson.ImageRef
				if len(imageref) > 0 {
					for i := range imageref {
						areas = append(areas, imageref[i])
						imageurns = append(imageurns, ctsurn)
					}
				}
				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				indexs = append(indexs, index)
			}

			return nil
		})
	}
	var correctedIndex []int
	k := 0
	for i := range indexs {
		if indexs[i] == 1 {
			k = i
		}
		result := k + indexs[i]
		correctedIndex = append(correctedIndex, result)
	}
	sort.Sort(dataframe{Indices: correctedIndex, Values1: texturns, Values2: texts})
	var content string
	content = "#!ctsdata\n"
	for i := range texturns {
		str := texturns[i] + "#" + texts[i] + "\n"
		content = content + str
	}
	content = content + "\n#!relations\n"
	for i := range imageurns {
		str := imageurns[i] + "#urn:cite2:dse:verbs.v1:appearsOn:#" + areas[i] + "\n"
		content = content + str
	}
	content = content + "\n"
	contentdispo := "Attachment; filename=" + filename + ".cex"
	modtime := time.Now()
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.Header().Add("Content-Disposition", contentdispo)
	http.ServeContent(w, r, filename, modtime, bytes.NewReader([]byte(content)))
}

func SaveImageRef(w http.ResponseWriter, r *http.Request) {

	//DEBUGGING
	fmt.Println(r.Method)
	if r.Method != "POST" {
		io.WriteString(w, "Only POST is supported!")
		return
	}
	fmt.Println(r.ParseForm())
	fmt.Println(r.FormValue("text"))

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("SaveImageRef", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	vars := mux.Vars(r)
	newkey := vars["key"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	imagerefstr := r.FormValue("text")
	imageref := strings.Split(imagerefstr, "#")
	dbname := user + ".db"
	retrieveddata := BoltRetrieve(dbname, newbucket, newkey)
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	fmt.Println(retrievedjson.ImageRef) //DEBUG
	retrievedjson.ImageRef = imageref
	fmt.Println(imageref) //DEBUG
	newnode, _ := json.Marshal(retrievedjson)
	db, err := OpenBoltDB(dbname)
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	key := []byte(newkey)    //
	value := []byte(newnode) //
	// store some data
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(newbucket))
		if err != nil {
			return err
		}

		err = bucket.Put(key, value)
		if err != nil {
			fmt.Println(err)
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
	http.Redirect(w, r, "/view/"+newkey, http.StatusFound)
}

func AddFirstNode(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("AddFirstNode", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	var texturns, texts, previouss, nexts, firsts, lasts []string
	var imagerefs, linetexts [][]string
	var indexs []int
	vars := mux.Vars(r)
	newkey := vars["key"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	dbname := user + ".db"
	retrieveddata := BoltRetrieve(dbname, newbucket, newkey)
	retrievednodejson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievednodejson)
	marker := retrievednodejson.First
	retrieveddata = BoltRetrieve(dbname, newbucket, marker)
	retrievednodejson = BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievednodejson)
	bookmark := retrievednodejson.Index
	lastnode := false
	if retrievednodejson.Last == retrievednodejson.URN {
		lastnode = true
	}
	db, err := OpenBoltDB(dbname)
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte(newbucket))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			retrievedjson := BoltURN{}
			json.Unmarshal([]byte(v), &retrievedjson)
			ctsurn := retrievedjson.URN
			text := retrievedjson.Text
			linetext := retrievedjson.LineText
			previous := retrievedjson.Previous
			next := retrievedjson.Next
			imageref := retrievedjson.ImageRef
			last := retrievedjson.Last
			index := retrievedjson.Index
			newfirst := newbucket + "newNode" + strconv.Itoa(bookmark)

			switch {
			case index < bookmark:
				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				linetexts = append(linetexts, linetext)
				previouss = append(previouss, previous)
				nexts = append(nexts, next)
				firsts = append(firsts, newfirst)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, index)
				imagerefs = append(imagerefs, imageref)
			case index > bookmark+1:
				newindex := index + 1
				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				linetexts = append(linetexts, linetext)
				previouss = append(previouss, previous)
				nexts = append(nexts, next)
				firsts = append(firsts, newfirst)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, newindex)
				imagerefs = append(imagerefs, imageref)
			case index == bookmark:
				newnode := newbucket + "newNode" + strconv.Itoa(index)
				newindex := index + 1

				texturns = append(texturns, newnode)
				texts = append(texts, "")
				linetexts = append(linetexts, []string{})
				previouss = append(previouss, newfirst)
				nexts = append(nexts, ctsurn)
				firsts = append(firsts, newfirst)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, index)
				imagerefs = append(imagerefs, []string{})

				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				linetexts = append(linetexts, linetext)
				previouss = append(previouss, newfirst)
				nexts = append(nexts, next)
				firsts = append(firsts, newfirst)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, newindex)
				imagerefs = append(imagerefs, imageref)
			case index == bookmark+1:
				newnode := newbucket + "newNode" + strconv.Itoa(bookmark)
				newindex := index + 1
				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				linetexts = append(linetexts, linetext)
				previouss = append(previouss, newnode)
				nexts = append(nexts, next)
				firsts = append(firsts, newfirst)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, newindex)
				imagerefs = append(imagerefs, imageref)
			}
		}

		return nil
	})

	var bolturns []BoltURN
	for i := range texturns {
		bolturns = append(bolturns, BoltURN{URN: texturns[i],
			Text:     texts[i],
			LineText: linetexts[i],
			Previous: previouss[i],
			Next:     nexts[i],
			First:    firsts[i],
			Last:     lasts[i],
			Index:    indexs[i],
			ImageRef: imagerefs[i]})
	}
	for i := range bolturns {
		newkey := texturns[i]
		newnode, _ := json.Marshal(bolturns[i])
		key := []byte(newkey)
		value := []byte(newnode)
		err = db.Update(func(tx *bolt.Tx) error {
			bucket, err := tx.CreateBucketIfNotExists([]byte(newbucket))
			if err != nil {
				return err
			}

			err = bucket.Put(key, value)
			if err != nil {
				return err
			}
			return nil
		})
	}
}

func AddNodeAfter(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("AddNodeAfter", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	var texturns, texts, previouss, nexts, firsts, lasts []string
	var imagerefs, linetexts [][]string
	var indexs []int
	vars := mux.Vars(r)
	newkey := vars["key"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"

	dbname := user + ".db"
	retrieveddata := BoltRetrieve(dbname, newbucket, newkey)
	retrievednodejson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievednodejson)
	bookmark := retrievednodejson.Index
	lastnode := false
	if retrievednodejson.Last == retrievednodejson.URN {
		lastnode = true
	}
	db, err := OpenBoltDB(dbname)
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte(newbucket))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			retrievedjson := BoltURN{}
			json.Unmarshal([]byte(v), &retrievedjson)
			ctsurn := retrievedjson.URN
			text := retrievedjson.Text
			linetext := retrievedjson.LineText
			previous := retrievedjson.Previous
			next := retrievedjson.Next
			first := retrievedjson.First
			imageref := retrievedjson.ImageRef
			last := retrievedjson.Last
			index := retrievedjson.Index

			switch {
			case index < bookmark:
				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				linetexts = append(linetexts, linetext)
				previouss = append(previouss, previous)
				nexts = append(nexts, next)
				firsts = append(firsts, first)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, index)
				imagerefs = append(imagerefs, imageref)
			case index > bookmark+1:
				newindex := index + 1
				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				linetexts = append(linetexts, linetext)
				previouss = append(previouss, previous)
				nexts = append(nexts, next)
				firsts = append(firsts, first)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, newindex)
				imagerefs = append(imagerefs, imageref)
			case index == bookmark:
				newnode := newbucket + "newNode" + strconv.Itoa(index)
				newindex := index + 1

				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				linetexts = append(linetexts, linetext)
				previouss = append(previouss, previous)
				nexts = append(nexts, newnode)
				firsts = append(firsts, first)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, index)
				imagerefs = append(imagerefs, imageref)

				texturns = append(texturns, newnode)
				texts = append(texts, "")
				linetexts = append(linetexts, []string{})
				previouss = append(previouss, ctsurn)
				nexts = append(nexts, next)
				firsts = append(firsts, first)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, newindex)
				imagerefs = append(imagerefs, []string{})
			case index == bookmark+1:
				newnode := newbucket + "newNode" + strconv.Itoa(bookmark)
				newindex := index + 1
				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				linetexts = append(linetexts, linetext)
				previouss = append(previouss, newnode)
				nexts = append(nexts, next)
				firsts = append(firsts, first)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, newindex)
				imagerefs = append(imagerefs, imageref)
			}
		}

		return nil
	})

	var bolturns []BoltURN
	for i := range texturns {
		bolturns = append(bolturns, BoltURN{URN: texturns[i],
			Text:     texts[i],
			LineText: linetexts[i],
			Previous: previouss[i],
			Next:     nexts[i],
			First:    firsts[i],
			Last:     lasts[i],
			Index:    indexs[i],
			ImageRef: imagerefs[i]})
	}
	for i := range bolturns {
		newkey := texturns[i]
		newnode, _ := json.Marshal(bolturns[i])
		key := []byte(newkey)
		value := []byte(newnode)
		err = db.Update(func(tx *bolt.Tx) error {
			bucket, err := tx.CreateBucketIfNotExists([]byte(newbucket))
			if err != nil {
				return err
			}

			err = bucket.Put(key, value)
			if err != nil {
				return err
			}
			return nil
		})
	}
}

func newText(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("newText", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	vars := mux.Vars(r)
	newkey := vars["key"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	dbname := user + ".db"
	retrievedjson := BoltURN{}
	retrievedjson.URN = newkey
	newnode, _ := json.Marshal(retrievedjson)
	db, err := OpenBoltDB(dbname)
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	key := []byte(newkey)    //
	value := []byte(newnode) //
	// store some data
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(newbucket))
		if err != nil {
			return err
		}

		err = bucket.Put(key, value)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
	http.Redirect(w, r, "/view/"+newkey, http.StatusFound)
}

func SaveTranscription(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("SaveTranscription", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	vars := mux.Vars(r)
	newkey := vars["key"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	text := r.FormValue("text")
	linetext := strings.Split(text, "\r\n")
	text = strings.Replace(text, "\r\n", "", -1)
	dbname := user + ".db"
	retrieveddata := BoltRetrieve(dbname, newbucket, newkey)
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	retrievedjson.Text = text
	retrievedjson.LineText = linetext
	newnode, _ := json.Marshal(retrievedjson)
	db, err := OpenBoltDB(dbname)
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	key := []byte(newkey)    //
	value := []byte(newnode) //
	// store some data
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(newbucket))
		if err != nil {
			return err
		}

		err = bucket.Put(key, value)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
	http.Redirect(w, r, "/view/"+newkey, http.StatusFound)
}

//LoadDB loads a CEX file, parses it, and saves its contents in the user DB.
//THIS NAME IS MISLEADING
func LoadDB(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("LoadDB", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	vars := mux.Vars(r)
	cex := vars["cex"]                               //get the name of the CEX file from URL
	http_req := config.Host + "/cex/" + cex + ".cex" //build the URL to pass to cexHandler
	data, _ := getContent(http_req)                  //get response data using getContent and cexHandler
	str := string(data)                              //make the response data a string
	var urns, areas []string
	var catalog []BoltCatalog

	//read in the relations of the CEX file cutting away all unnecessary signs
	if strings.Contains(str, "#!relations") {
		relations := strings.Split(str, "#!relations")[1]
		relations = strings.Split(relations, "#!")[0]
		re := regexp.MustCompile("(?m)[\r\n]*^//.*$")
		relations = re.ReplaceAllString(relations, "")

		reader := csv.NewReader(strings.NewReader(relations))
		reader.Comma = '#'
		reader.LazyQuotes = true
		reader.FieldsPerRecord = 3

		for {
			line, error := reader.Read()
			if error == io.EOF {
				break
			} else if error != nil {
				log.Fatal(error)
			}
			if strings.Contains(line[1], "appearsOn") {
				urns = append(urns, line[0])
				areas = append(areas, line[2])
			}
		}
	}

	if strings.Contains(str, "#!ctscatalog") {
		ctsCatalog := strings.Split(str, "#!ctscatalog")[1]
		ctsCatalog = strings.Split(ctsCatalog, "#!")[0]
		re := regexp.MustCompile("(?m)[\r\n]*^//.*$")
		ctsCatalog = re.ReplaceAllString(ctsCatalog, "")

		var caturns, catcits, catgrps, catwrks, catvers, catexpls, onlines, languages []string
		// var languages [][]string

		reader := csv.NewReader(strings.NewReader(ctsCatalog))
		reader.Comma = '#'
		reader.LazyQuotes = true
		reader.FieldsPerRecord = -1
		reader.TrimLeadingSpace = true

		for {
			line, error := reader.Read()
			if error == io.EOF {
				break
			} else if error != nil {
				log.Fatal(error)
			}

			switch {
			case len(line) == 8:
				if line[0] != "urn" {
					caturns = append(caturns, line[0])
					catcits = append(catcits, line[1])
					catgrps = append(catgrps, line[2])
					catwrks = append(catwrks, line[3])
					catvers = append(catvers, line[4])
					catexpls = append(catexpls, line[5])
					onlines = append(onlines, line[6])
					languages = append(languages, line[7])
				}
			case len(line) != 8:
				fmt.Println("Catalogue Data not well formatted")
			}
		}
		for j := range caturns {
			catalog = append(catalog, BoltCatalog{URN: caturns[j], Citation: catcits[j], GroupName: catgrps[j], WorkTitle: catwrks[j], VersionLabel: catvers[j], ExemplarLabel: catexpls[j], Online: onlines[j], Language: languages[j]})
		}
	}

	ctsdata := strings.Split(str, "#!ctsdata")[1]
	ctsdata = strings.Split(ctsdata, "#!")[0]
	re := regexp.MustCompile("(?m)[\r\n]*^//.*$")
	ctsdata = re.ReplaceAllString(ctsdata, "")

	reader := csv.NewReader(strings.NewReader(ctsdata))
	reader.Comma = '#'
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	var texturns, text []string

	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			fmt.Println(line)
			log.Fatal(error)
		}
		switch {
		case len(line) == 2:
			texturns = append(texturns, line[0])
			text = append(text, line[1])
		case len(line) > 2:
			texturns = append(texturns, line[0])
			var textstring string
			for j := 1; j < len(line); j++ {
				textstring = textstring + line[j]
			}
			text = append(text, textstring)
		case len(line) < 2:
			fmt.Println("Wrong line:", line)
		}
	}

	works := append([]string(nil), texturns...)
	for i := range texturns {
		works[i] = strings.Join(strings.Split(texturns[i], ":")[0:4], ":") + ":"
	}
	works = removeDuplicatesUnordered(works)
	var boltworks []BoltWork
	var sortedcatalog []BoltCatalog
	for i := range works {
		work := works[i]
		testexist := false
		for j := range catalog {
			if catalog[j].URN == work {
				sortedcatalog = append(sortedcatalog, catalog[j])
				testexist = true
			}
		}
		if testexist == false {
			fmt.Println(works[i], " has not catalog entry")
			sortedcatalog = append(sortedcatalog, BoltCatalog{})
		}

		var bolturns []BoltURN
		var boltkeys []string
		for j := range texturns {
			if strings.Contains(texturns[j], work) {
				var textareas []string
				if contains(urns, texturns[j]) {
					for k := range urns {
						if urns[k] == texturns[j] {
							textareas = append(textareas, areas[k])
						}
					}
				}
				linetext := strings.Split(text[j], "-NEWLINE-")
				bolturns = append(bolturns, BoltURN{URN: texturns[j], Text: text[j], LineText: linetext, ImageRef: textareas})
				boltkeys = append(boltkeys, texturns[j])
			}
		}
		for j := range bolturns {
			bolturns[j].Index = j + 1
			switch {
			case j+1 == len(bolturns):
				bolturns[j].Next = ""
			default:
				bolturns[j].Next = bolturns[j+1].URN
			}
			switch {
			case j == 0:
				bolturns[j].Previous = ""
			default:
				bolturns[j].Previous = bolturns[j-1].URN
			}
			bolturns[j].Last = bolturns[len(bolturns)-1].URN
			bolturns[j].First = bolturns[0].URN
		}
		boltworks = append(boltworks, BoltWork{Key: boltkeys, Data: bolturns})
	}
	boltdata := BoltData{Bucket: works, Data: boltworks, Catalog: sortedcatalog}

	// write to database
	pwd, _ := os.Getwd()
	dbname := pwd + "/" + user + ".db"
	db, err := OpenBoltDB(dbname)
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	for i := range boltdata.Bucket {
		newbucket := boltdata.Bucket[i]
		/// new stuff
		newcatkey := boltdata.Bucket[i]
		newcatnode, _ := json.Marshal(boltdata.Catalog[i])
		catkey := []byte(newcatkey)
		catvalue := []byte(newcatnode)
		err = db.Update(func(tx *bolt.Tx) error {
			bucket, err := tx.CreateBucketIfNotExists([]byte(newbucket))
			if err != nil {
				return err
			}

			err = bucket.Put(catkey, catvalue)
			if err != nil {
				return err
			}
			return nil
		})

		if err != nil {
			log.Fatal(err)
		}
		/// end stuff

		for j := range boltdata.Data[i].Key {
			newkey := boltdata.Data[i].Key[j]
			newnode, _ := json.Marshal(boltdata.Data[i].Data[j])
			key := []byte(newkey)
			value := []byte(newnode)
			// store some data
			err = db.Update(func(tx *bolt.Tx) error {
				bucket, err := tx.CreateBucketIfNotExists([]byte(newbucket))
				if err != nil {
					return err
				}

				err = bucket.Put(key, value)
				if err != nil {
					return err
				}
				return nil
			})

			if err != nil {
				log.Fatal(err)
			}
		}
	}
	io.WriteString(w, "Success")
	//This function should load a page using a template and display a propper success flash.
	//Alternatively it could become a helper function alltogether.
}

func fieldNWA(alntext []string) [][]string {
	letters := [][]string{}
	for i := range alntext {
		charSl := strings.Split(alntext[i], "")
		letters = append(letters, charSl)
	}
	length := len(letters)
	fields := make([][]string, length)
	tmp := make([]string, length)
	for i := range letters[0] {
		allspace := true
		for j := range letters {
			tmp[j] = tmp[j] + letters[j][i]
			if letters[j][i] != " " {
				allspace = false
			}
		}
		if allspace {
			for j := range letters {
				fields[j] = append(fields[j], tmp[j])
				tmp[j] = ""
			}
		}
	}
	for j := range letters {
		fields[j] = append(fields[j], tmp[j])
	}
	return fields
}

// func addSansHyphens(s string) string {
// 	hyphen := []rune(`&shy;`)
// 	after := []rune{rune('a'), rune('ā'), rune('i'), rune('ī'), rune('u'), rune('ū'), rune('ṛ'), rune('ṝ'), rune('ḷ'), rune('ḹ'), rune('e'), rune('o'), rune('ṃ'), rune('ḥ')}
// 	notBefore := []rune{rune('ṃ'), rune('ḥ'), rune(' ')}
// 	runeSl := []rune(s)
// 	newSl := []rune{}
// 	if len(runeSl) <= 2 {
// 		return s
// 	}
// 	newSl = append(newSl, runeSl[0:2]...)

// 	for i := 2; i < len(runeSl)-2; i++ {
// 		next := false
// 		possible := false
// 		for j := range after {
// 			if after[j] == runeSl[i] {
// 				possible = true
// 			}
// 		}
// 		if !possible {
// 			newSl = append(newSl, runeSl[i])
// 			continue
// 		}
// 		for j := range notBefore {
// 			if notBefore[j] == runeSl[i+1] {
// 				next = true
// 			}
// 		}
// 		if next {
// 			newSl = append(newSl, runeSl[i])
// 			next = false
// 			continue
// 		}
// 		if runeSl[i] == rune('a') {
// 			if runeSl[i+1] == rune('i') || runeSl[i+1] == rune('u') {
// 				newSl = append(newSl, runeSl[i])
// 				continue
// 			}
// 		}
// 		if runeSl[i-1] == rune(' ') {
// 			newSl = append(newSl, runeSl[i])
// 			continue
// 		}
// 		newSl = append(newSl, runeSl[i])
// 		for k := range hyphen {
// 			newSl = append(newSl, hyphen[k])
// 		}
// 	}
// 	newSl = append(newSl, runeSl[len(runeSl)-1:]...)
// 	return string(newSl)
// }

func findSpace(runeSl []rune) (spBefore, spAfter int, newSl []rune) {
	spAfter = 0
	spBefore = 0
	for i := 0; i < len(runeSl); i++ {
		if runeSl[i] == rune(' ') {
			spBefore++
		} else {
			break
		}
	}
	for i := len(runeSl) - 1; i >= 0; i-- {
		if runeSl[i] == rune(' ') {
			spAfter++
		} else {
			break
		}
	}
	return spBefore, spAfter, runeSl[spBefore : len(runeSl)-spAfter]
}

func addSansHyphens(s string) string {
	hyphen := []rune(`&shy;`)
	after := []rune{rune('a'), rune('ā'), rune('i'), rune('ī'), rune('u'), rune('ū'), rune('ṛ'), rune('ṝ'), rune('ḷ'), rune('ḹ'), rune('e'), rune('o'), rune('ṃ'), rune('ḥ')}
	notBefore := []rune{rune('ṃ'), rune('ḥ'), rune(' ')}
	runeSl := []rune(s)
	spBefore, spAfter, runeSl := findSpace(runeSl)
	newSl := []rune{}
	if len(runeSl) <= 2 {
		return s
	}
	newSl = append(newSl, runeSl[0:2]...)

	for i := 2; i < len(runeSl)-2; i++ {
		next := false
		possible := false
		for j := range after {
			if after[j] == runeSl[i] {
				possible = true
			}
		}
		if !possible {
			newSl = append(newSl, runeSl[i])
			continue
		}
		for j := range notBefore {
			if notBefore[j] == runeSl[i+1] {
				next = true
			}
		}
		if next {
			newSl = append(newSl, runeSl[i])
			next = false
			continue
		}
		if runeSl[i] == rune('a') {
			if runeSl[i+1] == rune('i') || runeSl[i+1] == rune('u') {
				newSl = append(newSl, runeSl[i])
				continue
			}
		}
		if runeSl[i-1] == rune(' ') {
			newSl = append(newSl, runeSl[i])
			continue
		}
		newSl = append(newSl, runeSl[i])
		for k := range hyphen {
			newSl = append(newSl, hyphen[k])
		}
	}
	SpBefore := []rune{}
	SpAfter := []rune{}
	for i := 0; i < spBefore; i++ {
		SpBefore = append(SpBefore, rune(' '))
	}
	for i := 0; i < spAfter; i++ {
		SpAfter = append(SpAfter, rune(' '))
	}
	if len(runeSl) < 4 {
		newSl = append(newSl, runeSl[len(runeSl)-1:]...)
	} else {
		newSl = append(newSl, runeSl[len(runeSl)-2:]...)
	}
	newSl = append(newSl, SpAfter...)
	newSl = append(SpBefore, newSl...)
	return string(newSl)
}

func nwa(text, text2 string) []string {
	hashreg := regexp.MustCompile(`#+`)
	punctreg := regexp.MustCompile(`[^\p{L}\s#]+`)
	swirlreg := regexp.MustCompile(`{[^}]*}`)
	text = swirlreg.ReplaceAllString(text, "")
	text2 = swirlreg.ReplaceAllString(text2, "")
	start := `<div class="tile is-child" lnum="L1">`
	start2 := `<div class="tile is-child" lnum="L2">`
	end := `</div>`
	collection := []string{text, text2}
	for i := range collection {
		collection[i] = strings.ToLower(collection[i])
	}
	var basetext []Word
	var comparetext []Word
	var highlight float32

	runealn1, runealn2, _ := gonwr.Align([]rune(collection[0]), []rune(collection[1]), rune('#'), 1, -1, -1)
	aln1 := string(runealn1)
	aln2 := string(runealn2)
	aligncol := fieldNWA([]string{aln1, aln2})
	aligned1, aligned2 := aligncol[0], aligncol[1]
	for i := range aligned1 {
		tmpA := hashreg.ReplaceAllString(aligned1[i], "")
		tmpB := hashreg.ReplaceAllString(aligned2[i], "")
		tmp2A := punctreg.ReplaceAllString(tmpA, "")
		tmp2B := punctreg.ReplaceAllString(tmpB, "")
		_, _, score := gonwr.Align([]rune(tmp2A), []rune(tmp2B), rune('#'), 1, -1, -1)
		base := len([]rune(tmpA))
		if len([]rune(tmpB)) > base {
			base = len([]rune(tmpB))
		}
		switch {
		case score <= 0:
			highlight = 1.0
		case score >= base:
			highlight = 0.0
		default:
			highlight = 1.0 - float32(score)/float32(base)
		}
		basetext = append(basetext, Word{Appearance: tmpA, Id: i + 1, Alignment: i + 1, Highlight: highlight})
		comparetext = append(comparetext, Word{Appearance: tmpB, Id: i + 1, Alignment: i + 1, Highlight: highlight})

	}
	text2 = start2
	for i := range comparetext {
		s := fmt.Sprintf("%.2f", comparetext[i].Highlight)
		switch comparetext[i].Id {
		case 0:
			text2 = text2 + "<span hyphens=\"manual\" style=\"background: rgba(255, 221, 87, " + s + ");\" id=\"" + strconv.Itoa(i+1) + "\" alignment=\"" + strconv.Itoa(comparetext[i].Alignment) + "\">" + addSansHyphens(comparetext[i].Appearance) + "</span>" + " "
		default:
			text2 = text2 + "<span hyphens=\"manual\" style=\"background: rgba(255, 221, 87, " + s + ");\" id=\"" + strconv.Itoa(i+1) + "\" alignment=\"" + strconv.Itoa(comparetext[i].Alignment) + "\">" + addSansHyphens(comparetext[i].Appearance) + "</span>" + " "
		}
	}
	text2 = text2 + end

	text = start
	for i := range basetext {
		s := fmt.Sprintf("%.2f", basetext[i].Highlight)
		for j := range comparetext {
			if comparetext[j].Alignment == basetext[i].Id {
				basetext[i].Alignment = comparetext[j].Id
			}
		}
		text = text + "<span hyphens=\"manual\" style=\"background: rgba(255, 221, 87, " + s + ");\" + id=\"" + strconv.Itoa(basetext[i].Id) + "\" alignment=\"" + strconv.Itoa(basetext[i].Alignment) + "\">" + addSansHyphens(basetext[i].Appearance) + "</span>" + " "
	}
	text = text + end

	return []string{text, text2}
}

func maxfloat(floatslice []float64) int {
	max := floatslice[0]
	maxindex := 0
	for i, value := range floatslice {
		if value > max {
			max = value
			maxindex = i
		}
	}
	return maxindex
}

// ViewPage generates the webpage based on the sent request
func ViewPage(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("ViewPage", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	vars := mux.Vars(r)
	urn := vars["urn"]
	dbname := user + ".db"

	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedcat := BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson := BoltCatalog{}
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	ctsurn := retrievedjson.URN
	text := "<p>"
	linetext := retrievedjson.LineText
	for i := range linetext {
		text = text + linetext[i]
		if i < len(linetext)-1 {
			text = text + "<br>"
		}
	}
	text = text + "</p>"
	previous := retrievedjson.Previous
	next := retrievedjson.Next
	imageref := retrievedjson.ImageRef
	first := retrievedjson.First
	last := retrievedjson.Last
	imagejs := "urn:cite2:test:googleart.positive:DuererHare1502"
	switch len(imageref) > 0 {
	case true:
		imagejs = imageref[0]
	}
	catid := retrievedcatjson.URN
	catcit := retrievedcatjson.Citation
	catgroup := retrievedcatjson.GroupName
	catwork := retrievedcatjson.WorkTitle
	catversion := retrievedcatjson.VersionLabel
	catexpl := retrievedcatjson.ExemplarLabel
	caton := retrievedcatjson.Online
	catlan := retrievedcatjson.Language

	transcription := Transcription{CTSURN: ctsurn,
		Transcriber:   user,
		Transcription: text,
		Previous:      previous,
		Next:          next,
		First:         first,
		Last:          last,
		TextRef:       textref,
		ImageRef:      imageref,
		ImageJS:       imagejs,
		CatID:         catid,
		CatCit:        catcit,
		CatGroup:      catgroup,
		CatWork:       catwork,
		CatVers:       catversion,
		CatExmpl:      catexpl,
		CatOn:         caton,
		CatLan:        catlan}

	kind := "/view/"
	p, _ := loadPage(transcription, kind)
	renderTemplate(w, "view", p)
}

func comparePage(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("comparePage", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	vars := mux.Vars(r)
	urn := vars["urn"]
	urn2 := vars["urn2"]
	dbname := user + ".db"

	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedcat := BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson := BoltCatalog{}
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	ctsurn := retrievedjson.URN
	text := ""
	linetext := retrievedjson.LineText
	for i := range linetext {
		text = text + linetext[i]
		if i < len(linetext)-1 {
			text = text + " "
		}
	}
	previous := retrievedjson.Previous
	next := retrievedjson.Next
	imageref := retrievedjson.ImageRef
	first := retrievedjson.First
	last := retrievedjson.Last
	imagejs := "urn:cite2:test:googleart.positive:DuererHare1502"
	switch len(imageref) > 0 {
	case true:
		imagejs = imageref[0]
	}
	catid := retrievedcatjson.URN
	catcit := retrievedcatjson.Citation
	catgroup := retrievedcatjson.GroupName
	catwork := retrievedcatjson.WorkTitle
	catversion := retrievedcatjson.VersionLabel
	catexpl := retrievedcatjson.ExemplarLabel
	caton := retrievedcatjson.Online
	catlan := retrievedcatjson.Language

	transcription := Transcription{CTSURN: ctsurn,
		Transcriber:   user,
		Transcription: text,
		Previous:      previous,
		Next:          next,
		First:         first,
		Last:          last,
		TextRef:       textref,
		ImageRef:      imageref,
		ImageJS:       imagejs,
		CatID:         catid,
		CatCit:        catcit,
		CatGroup:      catgroup,
		CatWork:       catwork,
		CatVers:       catversion,
		CatExmpl:      catexpl,
		CatOn:         caton,
		CatLan:        catlan}

	requestedbucket = strings.Join(strings.Split(urn2, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata = BoltRetrieve(dbname, requestedbucket, urn2)
	retrievedcat = BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson = BoltCatalog{}
	retrievedjson = BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	ctsurn = retrievedjson.URN
	text = ""
	linetext = retrievedjson.LineText
	for i := range linetext {
		text = text + linetext[i]
		if i < len(linetext)-1 {
			text = text + " "
		}
	}
	previous = retrievedjson.Previous
	next = retrievedjson.Next
	imageref = retrievedjson.ImageRef
	first = retrievedjson.First
	last = retrievedjson.Last
	imagejs = "urn:cite2:test:googleart.positive:DuererHare1502"
	switch len(imageref) > 0 {
	case true:
		imagejs = imageref[0]
	}
	catid = retrievedcatjson.URN
	catcit = retrievedcatjson.Citation
	catgroup = retrievedcatjson.GroupName
	catwork = retrievedcatjson.WorkTitle
	catversion = retrievedcatjson.VersionLabel
	catexpl = retrievedcatjson.ExemplarLabel
	caton = retrievedcatjson.Online
	catlan = retrievedcatjson.Language

	transcription2 := Transcription{CTSURN: ctsurn,
		Transcriber:   user,
		Transcription: text,
		Previous:      previous,
		Next:          next,
		First:         first,
		Last:          last,
		TextRef:       textref,
		ImageRef:      imageref,
		ImageJS:       imagejs,
		CatID:         catid,
		CatCit:        catcit,
		CatGroup:      catgroup,
		CatWork:       catwork,
		CatVers:       catversion,
		CatExmpl:      catexpl,
		CatOn:         caton,
		CatLan:        catlan}

	p, _ := loadCompPage(transcription, transcription2)
	renderCompTemplate(w, "compare", p)
}

func consolidatePage(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("consolidatePage", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	vars := mux.Vars(r)
	urn := vars["urn"]
	urn2 := vars["urn2"]
	dbname := user + ".db"

	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedcat := BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson := BoltCatalog{}
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	ctsurn := retrievedjson.URN
	text := ""
	linetext := retrievedjson.LineText
	for i := range linetext {
		text = text + linetext[i]
		if i < len(linetext)-1 {
			text = text + " "
		}
	}
	previous := retrievedjson.Previous
	next := retrievedjson.Next
	imageref := retrievedjson.ImageRef
	first := retrievedjson.First
	last := retrievedjson.Last
	imagejs := "urn:cite2:test:googleart.positive:DuererHare1502"
	switch len(imageref) > 0 {
	case true:
		imagejs = imageref[0]
	}
	catid := retrievedcatjson.URN
	catcit := retrievedcatjson.Citation
	catgroup := retrievedcatjson.GroupName
	catwork := retrievedcatjson.WorkTitle
	catversion := retrievedcatjson.VersionLabel
	catexpl := retrievedcatjson.ExemplarLabel
	caton := retrievedcatjson.Online
	catlan := retrievedcatjson.Language

	transcription := Transcription{CTSURN: ctsurn,
		Transcriber:   user,
		Transcription: text,
		Previous:      previous,
		Next:          next,
		First:         first,
		Last:          last,
		TextRef:       textref,
		ImageRef:      imageref,
		ImageJS:       imagejs,
		CatID:         catid,
		CatCit:        catcit,
		CatGroup:      catgroup,
		CatWork:       catwork,
		CatVers:       catversion,
		CatExmpl:      catexpl,
		CatOn:         caton,
		CatLan:        catlan}

	requestedbucket = strings.Join(strings.Split(urn2, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata = BoltRetrieve(dbname, requestedbucket, urn2)
	retrievedcat = BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson = BoltCatalog{}
	retrievedjson = BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	ctsurn = retrievedjson.URN
	text = ""
	linetext = retrievedjson.LineText
	for i := range linetext {
		text = text + linetext[i]
		if i < len(linetext)-1 {
			text = text + " "
		}
	}
	previous = retrievedjson.Previous
	next = retrievedjson.Next
	imageref = retrievedjson.ImageRef
	first = retrievedjson.First
	last = retrievedjson.Last
	imagejs = "urn:cite2:test:googleart.positive:DuererHare1502"
	switch len(imageref) > 0 {
	case true:
		imagejs = imageref[0]
	}
	catid = retrievedcatjson.URN
	catcit = retrievedcatjson.Citation
	catgroup = retrievedcatjson.GroupName
	catwork = retrievedcatjson.WorkTitle
	catversion = retrievedcatjson.VersionLabel
	catexpl = retrievedcatjson.ExemplarLabel
	caton = retrievedcatjson.Online
	catlan = retrievedcatjson.Language

	transcription2 := Transcription{CTSURN: ctsurn,
		Transcriber:   user,
		Transcription: text,
		Previous:      previous,
		Next:          next,
		First:         first,
		Last:          last,
		TextRef:       textref,
		ImageRef:      imageref,
		ImageJS:       imagejs,
		CatID:         catid,
		CatCit:        catcit,
		CatGroup:      catgroup,
		CatWork:       catwork,
		CatVers:       catversion,
		CatExmpl:      catexpl,
		CatOn:         caton,
		CatLan:        catlan}

	p, _ := loadCompPage(transcription, transcription2)
	renderCompTemplate(w, "consolidate", p)
}

//EditCatPage loads and renders the Edit Metadata page
func EditCatPage(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("EditCatPage", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	vars := mux.Vars(r)
	urn := vars["urn"]
	dbname := user + ".db"
	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedcat := BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson := BoltCatalog{}
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)
	previous := retrievedjson.Previous
	next := retrievedjson.Next
	first := retrievedjson.First
	last := retrievedjson.Last

	ctsurn := retrievedjson.URN
	catid := retrievedcatjson.URN
	catcit := retrievedcatjson.Citation
	catgroup := retrievedcatjson.GroupName
	catwork := retrievedcatjson.WorkTitle
	catversion := retrievedcatjson.VersionLabel
	catexpl := retrievedcatjson.ExemplarLabel
	caton := retrievedcatjson.Online
	catlan := retrievedcatjson.Language
	transcription := Transcription{CTSURN: ctsurn,
		Transcriber: user,
		TextRef:     textref,
		Previous:    previous,
		Next:        next,
		First:       first,
		Last:        last,
		CatID:       catid, CatCit: catcit, CatGroup: catgroup, CatWork: catwork, CatVers: catversion, CatExmpl: catexpl, CatOn: caton, CatLan: catlan}
	kind := "/editcat/"
	p, _ := loadPage(transcription, kind)
	renderTemplate(w, "editcat", p)
}

//EditPage loads and renders the Transcription Desk
func EditPage(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("EditPage", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	vars := mux.Vars(r)
	urn := vars["urn"]
	dbname := user + ".db"
	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)

	ctsurn := retrievedjson.URN
	linetext := retrievedjson.LineText
	previous := retrievedjson.Previous
	next := retrievedjson.Next
	imageref := retrievedjson.ImageRef
	first := retrievedjson.First
	last := retrievedjson.Last
	imagejs := "urn:cite2:test:googleart.positive:DuererHare1502"
	switch len(imageref) > 0 {
	case true:
		imagejs = imageref[0]
	}
	text := ""
	for i := range linetext {
		text = text + linetext[i]
		if i < len(linetext)-1 {
			text = text + "\r\n"
		}
	}
	transcription := Transcription{CTSURN: ctsurn,
		Transcriber:   user,
		Transcription: text,
		Previous:      previous,
		Next:          next,
		First:         first,
		Last:          last,
		TextRef:       textref,
		ImageRef:      imageref,
		ImageJS:       imagejs}
	kind := "/edit/"
	p, _ := loadPage(transcription, kind)
	renderTemplate(w, "edit", p)
}

//Edit2Page loads and renders the Image Citation Editor
func Edit2Page(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("Edit2Page", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	vars := mux.Vars(r)
	urn := vars["urn"]
	dbname := user + ".db"
	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)

	ctsurn := retrievedjson.URN
	text := retrievedjson.Text
	previous := retrievedjson.Previous
	next := retrievedjson.Next
	imageref := retrievedjson.ImageRef
	first := retrievedjson.First
	last := retrievedjson.Last
	imagejs := "urn:cite2:test:googleart.positive:DuererHare1502"
	switch len(imageref) > 0 {
	case true:
		imagejs = imageref[0]
	}
	transcription := Transcription{CTSURN: ctsurn,
		Transcriber:   user,
		Transcription: text,
		Previous:      previous,
		Next:          next,
		First:         first,
		Last:          last,
		TextRef:       textref,
		ImageRef:      imageref,
		ImageJS:       imagejs}
	kind := "/edit2/"
	p, _ := loadPage(transcription, kind)
	renderTemplate(w, "edit2", p)
}

// multi alignment testing

type Alignments struct {
	Alignment []Alignment
	Name      []string
}

type Alignment struct {
	Source []string
	Target []string
	Score  []float32
}

func MultiPage(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("MultiPage", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	vars := mux.Vars(r)
	urn := vars["urn"]

	dbname := user + ".db"

	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"
	work := strings.Join(strings.Split(strings.Split(requestedbucket, ":")[3], ".")[0:1], ".")
	retrieveddata := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	id1 := retrievedjson.URN
	text1 := retrievedjson.Text
	next1 := retrievedjson.Next
	first1 := retrievedjson.First
	last1 := retrievedjson.Last
	previous1 := retrievedjson.Previous
	swirlreg := regexp.MustCompile(`{[^}]*}`)
	text1 = swirlreg.ReplaceAllString(text1, "")
	text1 = strings.Replace(text1, "-NEWLINE-", "", -1)
	ids := []string{}
	texts := []string{}
	passageId := strings.Split(urn, ":")[4]

	buckets := Buckets(dbname)
	db, err := OpenBoltDB(dbname)
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for i := range buckets {
		if buckets[i] == requestedbucket {
			continue
		}
		if !gocite.IsCTSURN(buckets[i]) {
			continue
		}
		if strings.Join(strings.Split(strings.Split(buckets[i], ":")[3], ".")[0:1], ".") != work {
			continue
		}
		db.View(func(tx *bolt.Tx) error {
			// Assume bucket exists and has keys
			b := tx.Bucket([]byte(buckets[i]))

			c := b.Cursor()

			for k, v := c.First(); k != nil; k, v = c.Next() {
				retrievedjson := BoltURN{}
				json.Unmarshal([]byte(v), &retrievedjson)
				ctsurn := retrievedjson.URN
				text := strings.Replace(retrievedjson.Text, "-NEWLINE-", "", -1)
				if passageId != strings.Split(ctsurn, ":")[4] {
					continue
				}
				// make sure only witness that contain text are included
				if len(strings.Replace(text, " ", "", -1)) > 5 {
					ids = append(ids, ctsurn)
					texts = append(texts, text)
				}
			}

			return nil
		})
	}
	db.Close()

	alignments := nwa2(text1, id1, texts, ids)
	slsl := [][]string{}
	for i := range alignments.Alignment {
		slsl = append(slsl, alignments.Alignment[i].Source)
	}
	reordered, ok := testStringSl(slsl)
	if !ok {
		panic(ok)
	}
	for i := range alignments.Alignment {
		newset := reordered[i]
		newsource := []string{}
		newtarget := []string{}
		newscore := []float32{}
		for j := range newset {
			tmpstr := ""
			tmpstr2 := ""
			for _, v := range newset[j] {
				tmpstr = tmpstr + alignments.Alignment[i].Source[v]
				tmpstr2 = tmpstr2 + alignments.Alignment[i].Target[v]
			}
			newsource = append(newsource, tmpstr)
			newtarget = append(newtarget, tmpstr2)
			var highlight float32
			_, _, score := gonwr.Align([]rune(tmpstr), []rune(tmpstr2), rune('#'), 1, -1, -1)
			base := len([]rune(tmpstr))
			if len([]rune(tmpstr2)) > base {
				base = len([]rune(tmpstr2))
			}
			switch {
			case score <= 0:
				highlight = 1.0
			case score >= base:
				highlight = 0.0
			default:
				highlight = 1.0 - float32(score)/float32(base)
			}
			newscore = append(newscore, highlight)
		}
		alignments.Alignment[i].Score = newscore
		alignments.Alignment[i].Source = newsource
		alignments.Alignment[i].Target = newtarget
	}
	start := `<div class="tile is-child" lnum="L`
	start1 := `<div id="`
	start2 := `" class="tile is-child" lnum="L`
	end := `</div>`
	tmpsl := []string{}
	tmpstr := start + strconv.Itoa(1) + `">`
	tmpstr2 := `<div class="items2">`

	for j, v := range alignments.Alignment[0].Source {
		var sc float32
		tmpstr2 = tmpstr2 + `<div id="crit` + strconv.Itoa(j+1) + `" class="content" style="display:none;">`
		appcrit := make(map[string]string)
		for k := range alignments.Alignment {
			sc = sc + alignments.Alignment[k].Score[j]
			if alignments.Alignment[k].Score[j] > float32(0) {
				newid := strings.Split(ids[k], ":")[3]
				newid = strings.Split(newid, ".")[2]
				item := alignments.Alignment[k].Target[j]
				newvalue := appcrit[item]
				if newvalue == "" {
					newvalue = newvalue + newid
				} else {
					newvalue = newvalue + "," + newid
				}
				appcrit[item] = newvalue
			}
		}
		appcount := 1
		for key, value := range appcrit {
			tmpstr2 = tmpstr2 + strconv.Itoa(appcount) + "."
			valueSl := strings.Split(value, ",")
			for _, valui := range valueSl {
				tmpstr2 = tmpstr2 + `<a href="#` + valui + `" onclick="highlfunc(this);">` + valui + `</a> `
			}
			tmpstr2 = tmpstr2 + addSansHyphens(key) + `<br/>`
			appcount++
		}
		tmpstr2 = tmpstr2 + end
		sc = sc / float32(len(alignments.Alignment))
		s := fmt.Sprintf("%.2f", sc)
		tmpstr = tmpstr + "<span hyphens=\"manual\" style=\"background: rgba(255, 221, 87, " + s + ");\" id=\"" + strconv.Itoa(j+1) + "\" alignment=\"" + strconv.Itoa(j+1) + "\">" + addSansHyphens(v) + "</span>" + " "
	}
	tmpstr2 = tmpstr2 + end
	tmpstr = tmpstr + end
	tmpsl = append(tmpsl, tmpstr)
	for i := range alignments.Alignment {
		newid := strings.Split(ids[i], ":")[3]
		newid = strings.Split(newid, ".")[2]
		tmpstr := start1 + newid + start2 + strconv.Itoa(i+2) + `">`
		for j, v := range alignments.Alignment[i].Target {
			s := fmt.Sprintf("%.2f", alignments.Alignment[i].Score[j])
			tmpstr = tmpstr + "<span hyphens=\"manual\" style=\"background: rgba(165, 204, 107, " + s + ");\" id=\"" + strconv.Itoa(j+1) + "\" alignment=\"" + strconv.Itoa(j+1) + "\">" + addSansHyphens(v) + "</span>" + " "
		}
		tmpstr = tmpstr + end
		tmpsl = append(tmpsl, tmpstr)
	}

	tmpstr = `<div class="tile is-ancestor"><div class="tile is-parent column is-6"><div class="container"><div class="card is-fullwidth"><header class="card-header"><p class="card-header-title">Text</p></header><div class="card-content"><div class="content">`
	tmpstr = tmpstr + tmpsl[0]
	tmpstr = tmpstr + end
	tmpstr = tmpstr + end
	tmpstr = tmpstr + end
	tmpstr = tmpstr + end
	tmpstr = tmpstr + end
	tmpstr = tmpstr + `<div class="tile is-parent column is-6"><div class="container"><div id="trmenu">`
	for _, v := range ids {
		newid := strings.Split(v, ":")[3]
		newid = strings.Split(newid, ".")[2]
		tmpstr = tmpstr + `<a class="button" id="button_` + newid + `" href="#` + newid + `" onclick="highlfunc(this);">` + newid + `</a>`
	}
	tmpstr = tmpstr + end
	tmpstr = tmpstr + `<div class="items">`
	for i, v := range tmpsl {
		if i == 0 {
			continue
		}
		tmpstr = tmpstr + v
	}
	tmpstr = tmpstr + end
	tmpstr = tmpstr + end
	tmpstr = tmpstr + end
	tmpstr = tmpstr + end

	tmpstr = tmpstr + `<div class="tile is-ancestor"><div class="tile is-parent column is-6"><div class="container"><div class="card"><header class="card-header"><p class="card-header-title">Variants</p></header><div class="card-content">` + tmpstr2 + end + end + end + end + end
	transcription := Transcription{
		CTSURN:        urn,
		Transcriber:   user,
		TextRef:       buckets,
		Next:          next1,
		Previous:      previous1,
		First:         first1,
		Last:          last1,
		Transcription: tmpstr}
	p, _ := loadMultiPage(transcription)
	renderTemplate(w, "multicompare", p)
}

func fieldNWA2(alntext []string) [][]string {
	letters := [][]string{}
	for i := range alntext {
		charSl := strings.Split(alntext[i], "")
		letters = append(letters, charSl)
	}
	length := len(letters)
	fields := make([][]string, length)
	tmp := make([]string, length)
	for i := range letters[0] {
		allspace := true
		for j := range letters {
			tmp[j] = tmp[j] + letters[j][i]
			if letters[j][i] != " " {
				allspace = false
			}
		}
		if allspace {
			for j := range letters {
				fields[j] = append(fields[j], tmp[j])
				tmp[j] = ""
			}
		}
	}
	for j := range letters {
		fields[j] = append(fields[j], tmp[j])
	}
	for i := range fields {
		fields[i][0] = strings.TrimLeft(fields[i][0], " ")
	}
	return fields
}

func nwa2(basetext, baseid string, texts, ids []string) (alignments Alignments) {
	hashreg := regexp.MustCompile(`#+`)
	punctreg := regexp.MustCompile(`[^\p{L}\s#]+`)
	swirlreg := regexp.MustCompile(`{[^}]*}`)
	var highlight float32

	for i := range texts {
		alignment := Alignment{}
		texts[i] = strings.ToLower(texts[i])
		texts[i] = strings.TrimSpace(texts[i])
		texts[i] = swirlreg.ReplaceAllString(texts[i], "")
		runealn1, runealn2, _ := gonwr.Align([]rune(basetext), []rune(texts[i]), rune('#'), 1, -1, -1)
		aln1 := string(runealn1)
		aln2 := string(runealn2)
		aligncol := fieldNWA2([]string{aln1, aln2})
		aligned1, aligned2 := aligncol[0], aligncol[1]
		for j := range aligned1 {
			tmpA := hashreg.ReplaceAllString(aligned1[j], "")
			tmpB := hashreg.ReplaceAllString(aligned2[j], "")
			tmp2A := punctreg.ReplaceAllString(tmpA, "")
			tmp2B := punctreg.ReplaceAllString(tmpB, "")
			_, _, score := gonwr.Align([]rune(tmp2A), []rune(tmp2B), rune('#'), 1, -1, -1)
			base := len([]rune(tmpA))
			if len([]rune(tmpB)) > base {
				base = len([]rune(tmpB))
			}
			switch {
			case score <= 0:
				highlight = 1.0
			case score >= base:
				highlight = 0.0
			default:
				highlight = 1.0 - float32(score)/float32(base)
			}
			alignment.Source = append(alignment.Source, tmpA)
			alignment.Target = append(alignment.Target, tmpB)
			alignment.Score = append(alignment.Score, highlight)
		}
		newID := baseid + "+" + ids[i]
		alignments.Name = append(alignments.Name, newID)
		alignments.Alignment = append(alignments.Alignment, alignment)
	}
	return alignments
}

func testString(str string, strsl1 []string, cursorIn int) (cursorOut int, sl []int, ok bool) {
	calcStr1 := ""
	if len([]rune(str)) > len([]rune(strings.Join(strsl1[cursorIn:], ""))) {
		return 0, []int{}, false
	}
	base := cursorIn
	for i, v := range strsl1[cursorIn:] {
		calcStr1 = calcStr1 + v
		if calcStr1 != str {
			if i+1 == len(sl) {
				return 0, []int{}, false
			}
			sl = append(sl, i+base)
			continue
		}
		if calcStr1 == str {
			sl = append(sl, i+base)
			cursorOut = i + base + 1
			ok = true
			return cursorOut, sl, ok
		}
	}
	return 0, []int{}, false
}

func testAllTheSame(testset [][]string) bool {
	teststr := strings.Join(testset[0], "")
	for i := range testset {
		if i == 0 {
			continue
		}
		if teststr != strings.Join(testset[i], "") {
			return false
		}
	}
	return true
}

func testStringSl(slsl [][]string) (slsl2 [][][]int, ok bool) {
	if len(slsl) == 0 {
		// fmt.Println("zero length")
		slsl2 = [][][]int{}
		return slsl2, ok
	}
	ok = testAllTheSame(slsl)
	if !ok {
		// fmt.Println("slices not same length")
		slsl2 = [][][]int{}
		return slsl2, ok
	}
	// fmt.Println("passed testAllTheSame")

	length := len(slsl)

	base := make([]int, length)
	cursor := make([]int, length)
	indeces := make([][]int, length)
	testr := ""
	slsl2 = make([][][]int, length)

	for i := 0; i < len(slsl[0]); i++ {
		match := false
		indeces[0] = append(indeces[0], i)
		testr = testr + slsl[0][i]
		// fmt.Println("test", testr)
		// fmt.Scanln()

		for k := range slsl {
			if k == 0 {
				continue
			}
			cursor[k], indeces[k], match = testString(testr, slsl[k], base[k])
			if !match {
				// fmt.Println(testr, "and", slsl[k][base[k]:], "do not match")
				// fmt.Scanln()
				break
			}
		}
		if match {
			// fmt.Println("write to slice!!")
			// fmt.Scanln()
			for k := range slsl {
				slsl2[k] = append(slsl2[k], indeces[k])
				if k == 0 {
					continue
				}
				base[k] = cursor[k]
			}
			indeces[0] = []int{}
			testr = ""
		}
	}
	ok = true
	return slsl2, ok
}

// old stuff

// 	calcStr1 := ""
// 	calcStr2 := ""
// 	tmpstr := ""
// 	accessed := false
// 	count := 0

// 	length := len(slsl)

// 	base := make([]int, length)
// 	cursor := make([]int, length)
// 	indeces := make([][]int, length)
// 	slsl2 = make([][][]int, length)

// 	for i, v := range slsl[0][base[0]:] {
// 		match := false
// 		smaller := false
// 		calcStr1 = calcStr1 + v
// 		if len([]rune(calcStr1)) < len([]rune(calcStr2)) {
// 			cursor[0]++
// 			indeces[0] = append(indeces[0], i)
// 			continue
// 		}
// 		for j, w := range slsl[1][base[1]:] {
// 			tmpstr = calcStr2
// 			calcStr2 = calcStr2 + w
// 			fmt.Println("compare", calcStr1, "and", calcStr2)
// 			fmt.Scanln()
// 			if len([]rune(calcStr1)) < len([]rune(calcStr2)) {
// 				smaller = true
// 				if accessed {
// 					calcStr2 = tmpstr
// 					// accessed = false
// 				} else {
// 					calcStr2 = ""
// 				}

// 				break
// 			}
// 			if len([]rune(calcStr1)) > len([]rune(calcStr2)) {
// 				fmt.Println(len([]rune(calcStr1)), "and", len([]rune(calcStr2)))
// 				fmt.Scanln()
// 				cursor[1]++
// 				count++
// 				indeces[1] = append(indeces[1], j+base[1])
// 				continue
// 			}
// 			if calcStr1 == calcStr2 {
// 				fmt.Println("strings 1 + 2 match")
// 				fmt.Scanln()
// 				for k := range slsl {
// 					if k < 2 {
// 						continue
// 					}
// 					cursor[k], indeces[k], match = testString(calcStr1, slsl[k], base[k])
// 					if !match {
// 						fmt.Println(calcStr1, "and", slsl[k][base[k]:], "do not match")
// 						fmt.Scanln()
// 						accessed = true
// 						break
// 					}

// 				}

// 				indeces[0] = append(indeces[0], i)
// 				indeces[1] = append(indeces[1], j+base[1])
// 				cursor[1]++
// 				cursor[0]++
// 				count = 0
// 				base[1] = cursor[1]
// 				base[0] = cursor[0]
// 				break
// 			}
// 			break
// 		}
// 		if smaller {
// 			fmt.Println("smaller. count:", count)
// 			cursor[0]++
// 			cursor[1] = cursor[1] - count
// 			base[1] = cursor[1]
// 			fmt.Println("restart with", slsl[1][base[1]])
// 			fmt.Scanln()
// 			count = 0
// 			indeces[1] = []int{}
// 			indeces[0] = append(indeces[0], i)
// 			continue
// 		}
// 		if match {
// 			fmt.Println("write to slice!!")
// 			fmt.Scanln()
// 			accessed = false
// 			count = 0
// 			for k := range slsl {
// 				slsl2[k] = append(slsl2[k], indeces[k])
// 				if k < 2 {
// 					continue
// 				}
// 				base[k] = cursor[k]
// 			}

// 			indeces[0] = []int{}
// 			indeces[1] = []int{}
// 			calcStr1 = ""
// 			calcStr2 = ""

// 			if base[0] == len(slsl[0]) {
// 				ok = true
// 				return slsl2, ok
// 			}
// 			continue
// 		}

// 	}
// 	fmt.Println("!!! accessed this!!")
// 	fmt.Scanln()
// 	ok = false
// 	for k := range slsl {
// 		slsl2[k] = [][]int{}
// 	}
// 	return slsl2, ok
// }
