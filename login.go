package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	//for using external login providers
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/gitlab"
	"github.com/markbates/goth/providers/google"

	"github.com/boltdb/bolt"

	"github.com/gorilla/sessions" //for Cookiestore and other session functionality
)

//BrucheionStore represents the session on the server side
var BrucheionStore sessions.Store

//SessionName saves the name of the Brucheion session
const SessionName = "brucheionSession"

//SetUpGothic sets up Gothic for login procedure
func setUpGothic() {
	//Build the authentification paths for the choosen providers
	gitHubPath := config.Host + "/auth/github/callback"
	gitLabPath := config.Host + "/auth/gitlab/callback"
	googlePath := config.Host + "/auth/google/callback"
	//Tell gothic which login providers to use
	goth.UseProviders(
		github.New(providers.GitHub.Key, providers.GitHub.Secret, gitHubPath),
		gitlab.New(providers.GitLab.Key, providers.GitLab.Secret, gitLabPath, "read_user"),
		google.New(providers.Google.Key, providers.Google.Secret, googlePath, "profile"))
	//Create new Cookiestore for _gothic_session
	loginTimeout := 60 //Time the _gothic_session cookie will be alive in seconds
	gothic.Store = getCookieStore(loginTimeout)
}

// loginGET renders the login page. The user can enter the login Credentials into the form.
//If already logged in, the user will be redirected to main page.
func loginGET(res http.ResponseWriter, req *http.Request) {
	//Make sure user is not logged in yet
	session, err := getSession(req) //Get a session
	if err != nil {
		log.Println(fmt.Errorf("loginGET: Error getting session: %s", err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if session.Values["Loggedin"] != nil { //test if the Loggedin variable has already been set
		//if session.Values["Loggedin"].(bool) { //"Loggedin" will be true if user is already logged in
		user, ok := session.Values["BrucheionUserName"].(string) //if there is a session get the username
		if !ok {
			log.Println("func loginGET: Type assertion to string failed for session value BrucheionUser or session value could not be retrieved.")
		}
		log.Printf("loginGET: user %s is already logged in. Redirecting to main\n", user)
		http.Redirect(res, req, "/main/", http.StatusFound)
		return
		//}
	} //Destroy the session we just got (proceed with login process)
	log.Println("loginGET: Session seems empty. Destroying session.")
	inSituLogout(res, req)

	loginPage := &LoginPage{
		Title:  "Brucheion Login Page",
		NoAuth: *noAuth}
	renderLoginTemplate(res, "login", loginPage)
}

//loginPOST logs in the user using the form values and gothic. //Todo: make better explanation
func loginPOST(res http.ResponseWriter, req *http.Request) {

	//Make sure user is not logged in yet
	session, err := getSession(req) //get a session
	if err != nil {
		log.Println(fmt.Errorf("loginPOST: Error getting session: %s", err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if session.Values["Loggedin"] != nil { //If the Loggedin variable has already been set..
		if session.Values["Loggedin"].(bool) { //And if "Loggedin" is true..
			user, ok := session.Values["BrucheionUserName"].(string) //Then get the username
			if !ok {
				fmt.Println("func loginPOST: Type assertion to string failed for session value BrucheionUser or session value could not be retrieved.")
			}
			log.Printf("loginPOST: User %s is already logged in. Redirecting to main\n", user) //Log that session was already logged in
			http.Redirect(res, req, "/main/", http.StatusFound)                                //redirect to main, as login is not necessary anymore
			return
		}
		//if not its an unknown session state, destroy session and redirect to login
		Logout(res, req)
		return
	}
	log.Println("loginPOST: Session seems empty. Destroying session.")
	//if user was not logged in yet then destroy the session we just got (and proceed with login process)
	inSituLogout(res, req)

	//populates Loginpage with basic data and the form values
	lp := &LoginPage{
		BUserName: strings.TrimSpace(req.FormValue("brucheionusername")),
		Host:      config.Host,
		Title:     "Brucheion Login Page", //set the title of the page
		NoAuth:    *noAuth}

	unameValidation := validateUserName(lp.BUserName) //checks if this username only has (latin) letters and (arabian) numbers

	if unameValidation.ErrorCode { //if a valid username has been chosen
		session, err = initializeSession(req) //initialize a persisting session
		if err != nil {
			fmt.Println("loginPOST: Error initializing the session.")
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		//save the BrucheionUserName in the session as well and set the Loggedin value to false
		session.Values["BrucheionUserName"] = lp.BUserName
		session.Values["Provider"] = req.FormValue("provider") //save the provider used for login in the session
		session.Values["Loggedin"] = false
		session.Save(req, res)

		err = initializeUsersDB() //Make sure the users.db file is there and has the necessary buckets.
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		if *noAuth { //if the noauth flag was set true: check if username is not in use for a login with a provider
			//log.Println("noAuth flag was true")

			validation, err := validateNoAuthUser(req) //validate if credentials match existing user and not in use with a provider login yet
			if err != nil {
				fmt.Printf("\nloginPOST error validating user: %s", err)
				http.Error(res, err.Error(), http.StatusInternalServerError)
			}
			if validation.ErrorCode { //if the username is valid
				brucheionUser := &BrucheionUser{ //create Brucheionuser instance
					BUserName: lp.BUserName,
					Provider:  "noAuth"}
				if !validation.BUserInUse { // create new noAuth user if the username was not in use
					db, err := openBoltDB(config.UserDB) //open bolt DB using helper function
					if err != nil {
						log.Println(fmt.Printf("loginPOST: error opening userDB: %s", err))
						http.Error(res, err.Error(), http.StatusInternalServerError)
						return
					}
					db.Update(func(tx *bolt.Tx) error {
						bucket := tx.Bucket([]byte("users"))
						buffer, err := json.Marshal(brucheionUser) //Marshal user data
						if err != nil {
							return fmt.Errorf("failed marshalling user data for user %s: %s", brucheionUser.BUserName, err)
						}
						err = bucket.Put([]byte(brucheionUser.BUserName), buffer) //put user into bucket
						if err != nil {
							return fmt.Errorf("failed saving user %s in users.db: %s", brucheionUser.BUserName, err)
						}
						log.Printf("Successfully saved new user %s in users.DB.\n", brucheionUser.BUserName)

						log.Println(validation.Message) //Display validation.Message if all went well.
						return nil
					})
					db.Close() //always remember to close the database
				}
				session.Values["Loggedin"] = true //To keep the user logged in
				session.Save(req, res)
				lp.Message = validation.Message //The message to be replied in regard to the login scenario
				renderAuthTemplate(res, "callback", lp)
				return
			}
			//if the username is not valid
			inSituLogout(res, req)                //kill the session
			lp.Message = validation.Message       //add the message to the loginpage
			renderLoginTemplate(res, "login", lp) //and render the login template again, displaying said message.
			return

		}
		//if the noauth flag was not set, or set false: continue with authentification using a provider
		log.Println("loginPost: validating if credentials match existing user")
		validation, err := validateUser(req) //validate if credentials match existing user
		if err != nil {
			fmt.Printf("\nLoginPost: error validating user: %s", err)
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		if validation.ErrorCode { //Login scenarios 1 (user in DB, should be able to login), (5) (user not in DB and not in use yet, shoould be able to register)
			authPath := "/auth/" + strings.ToLower(req.FormValue("provider")) + "/" //set up the path for redirect according to provider (needs to be lower case for gothic)
			session.Save(req, res)                                                  //always save the session after setting values
			http.Redirect(res, req, authPath, http.StatusFound)                     //redirect to auth page with correct provider
			return
		}

		log.Println(validation.Message)
		validation.Message = validation.Message + "\nPlease always use the same combination of username, provider, and provider account."
		if (!validation.BUserInUse) || (validation.BUserInUse && !validation.SameProvider && validation.PUserInUse) { //unknown login behavior
			log.Println("Unknown login behavior. This should never happen. Logging out.")
			validation.Message = "Unknown login behavior. Please report this to the development team."
		}
		//Login scenarios (2), (3), (4)
		lp := &LoginPage{
			Message: validation.Message}
		inSituLogout(res, req)
		renderLoginTemplate(res, "login", lp)
		return
	}
	//if the the user name was not valid
	lp.Message = unameValidation.Message  //add the message to the loginpage
	renderLoginTemplate(res, "login", lp) //and render the login template again, displaying said message.
}

//auth redirects to provider for authentification using gothic.
//Provider redirects to callback page.
func auth(res http.ResponseWriter, req *http.Request) {
	//Make sure user is not logged in yet
	session, err := getSession(req) //get a session
	if err != nil {
		log.Println(fmt.Errorf("Auth: Error getting session: %s", err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if session.Values["Loggedin"] != nil { //If the Loggedin variable has already been set
		if session.Values["Loggedin"].(bool) { //And if "Loggedin" is true
			user, ok := session.Values["BrucheionUserName"].(string) //Then get the username
			if !ok {
				fmt.Println("func Auth: Type assertion to string failed for session value BrucheionUser or session value could not be retrieved.")
			}
			log.Printf("func Auth: user %s is already logged in. Redirecting to main\n", user) //Log that session was already logged in
			http.Redirect(res, req, "/main/", http.StatusFound)                                //redirect to main, as login is not necessary anymore
			return
		}
		//proceed with login process (gothic redirects to provider and redirects to callback)
		gothic.BeginAuthHandler(res, req)
	} else { //kill the session and redirect to login
		log.Println("func Auth: \"Loggedin\" was nil. Session was not initialized. Logging out")
		Logout(res, req)
		return
	}
}

//authCallback completes user authentification, sets session variables and DB entries.
func authCallback(res http.ResponseWriter, req *http.Request) {
	//Make sure user is not logged in yet
	session, err := getSession(req) //get a session
	if err != nil {
		log.Println(fmt.Errorf("func authCallback: Error getting session: %s", err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if session.Values["Loggedin"] != nil { //If the Loggedin variable has already been set
		if session.Values["Loggedin"].(bool) { //And if "Loggedin" is true
			user, ok := session.Values["BrucheionUserName"].(string) //Then get the username
			if !ok {
				log.Println("func authCallback: Type assertion to string failed for session value BrucheionUser or session value could not be retrieved.")
			}
			log.Printf("authCallback: User %s is already logged in. Redirecting to main\n", user) //Log that this session was already logged in
			http.Redirect(res, req, "/main/", http.StatusFound)                                   //redirect to main, as login is not necessary anymore
			return
		} //else proceed with login process
	} else { //kill the session and redirect to login
		log.Println("func authCallback: \"Loggedin\" was nil. Session was not initialized.")
		Logout(res, req)
		return
	}

	//get the provider user from Gothic
	gothUser, err := gothic.CompleteUserAuth(res, req) //authentificate user and get gothUser from gothic
	if err != nil {
		fmt.Fprintln(res, err)
		return
	}

	//get provider and username from session values
	provider, ok := session.Values["Provider"].(string)
	if !ok {
		fmt.Println("Func authCallback: Type assertion of value Provider to string failed or session value could not be retrieved.")
	}
	brucheionUserName, ok := session.Values["BrucheionUserName"].(string)
	if !ok {
		fmt.Println("Func authCallback: Type assertion of value BrucheionUserName to string failed or session value could not be retrieved.")
	}

	//save values retrieved from gothUser in session
	session.Values["Loggedin"] = false                     //assumed for later use, maybe going to be deprecated later
	session.Values["ProviderNickName"] = gothUser.NickName //The nickname used for logging in with provider
	session.Values["ProviderUserID"] = gothUser.UserID     //the userID returned by the login from provider
	session.Save(req, res)                                 //always remember to save the session

	brucheionUser := &BrucheionUser{ //create Brucheionuser instance
		BUserName:      brucheionUserName,
		Provider:       provider,
		PUserName:      gothUser.NickName,
		ProviderUserID: gothUser.UserID}

	log.Println("func authCallback validating if credentials match existing user")
	validation, err := validateUser(req) //validate if credentials match existing user
	if err != nil {
		fmt.Printf("\nauthCallback error validating user: %s", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

	//Save user in DB and/or login user if user is valid. Redirect back to login page if not
	if validation.BUserInUse && validation.SameProvider && validation.PUserInUse { //Login scenario (1)
		session.Values["Loggedin"] = true
		session.Save(req, res)
		log.Println(validation.Message) //Display validation.Message if all went well.
	} else if !validation.BUserInUse && !validation.SameProvider && !validation.PUserInUse { //Login scenario (5)
		//create new entry for new BUser
		db, err := openBoltDB(config.UserDB) //open bolt DB using helper function
		if err != nil {
			log.Println(fmt.Printf("authCallback: error opening userDB: %s", err))
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte("users"))
			buffer, err := json.Marshal(brucheionUser) //Marshal user data
			if err != nil {
				return fmt.Errorf("failed marshalling user data for user %s: %s", brucheionUserName, err)
			}
			err = bucket.Put([]byte(brucheionUserName), buffer) //put user into bucket
			if err != nil {
				return fmt.Errorf("failed saving user %s in users.db: %s", brucheionUserName, err)
			}
			log.Printf("Successfully saved new user %s in users.DB.\n", brucheionUserName)

			bucket = tx.Bucket([]byte(provider))
			err = bucket.Put([]byte(brucheionUser.ProviderUserID), []byte(brucheionUserName))
			if err != nil {
				return fmt.Errorf("failed saving user ProviderUserID for user %s in Bucket %s: %s", brucheionUserName, provider, err)
			}
			log.Printf("Successfully saved ProviderUserID of BUser %s in Bucket %s.\n", brucheionUserName, provider)
			log.Println(validation.Message) //Display validation.Message if all went well.
			return nil
		})
		db.Close()                        //always remember to close the db
		session.Values["Loggedin"] = true //To keep the user logged in
		session.Save(req, res)

	} else if !validation.BUserInUse && !validation.SameProvider && validation.PUserInUse {
		log.Println(validation.Message)
		inSituLogout(res, req)
		lp := &LoginPage{Message: validation.Message} //add the message to the loginpage
		renderLoginTemplate(res, "login", lp)         //and render the login template again, displaying said message.
		return
	} else { //unknown login behavior
		log.Println("Unknown login behavior. This should never happen. Logging out.")
		Logout(res, req)
		return
	}

	lp := &LoginPage{
		Host:         config.Host,
		BUserName:    brucheionUserName,
		Provider:     provider,
		HrefUserName: brucheionUserName + "_" + provider,
		Message:      validation.Message} //The message to be replied in regard to the login scenario
	renderAuthTemplate(res, "callback", lp)
}

//Logout kills the session (equivalent to logging out), logs the logout, and redirects to login page.
func Logout(res http.ResponseWriter, req *http.Request) {

	session, err := getSession(req)
	if err != nil {
		log.Println("No session, no logout")
		return
	}

	bUserName, ok := session.Values["BrucheionUserName"].(string)
	if !ok {
		log.Println("Logout: BrucheionUserName could not be retrieved from session.")
	}

	session.Options.MaxAge = -1
	session.Values = make(map[interface{}]interface{})
	err = session.Save(req, res)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if bUserName == "" {
		log.Println("Logout: Empty session destroyed.")
	} else {
		log.Printf("Logout: User %s logged out\n", bUserName)
	}

	http.Redirect(res, req, "/login/", http.StatusFound)
}

//inSituLogout kills the session and does not redirects afterwards
func inSituLogout(res http.ResponseWriter, req *http.Request) {

	session, err := getSession(req)
	if err != nil {
		log.Println("Did not get a session, nothing to logout from")
		return
	}

	bUserName, ok := session.Values["BrucheionUserName"].(string)
	if !ok {
		log.Println("inSituLogout: No BrucheionUserName retrieved from session.")
	}

	session.Options.MaxAge = -1
	session.Values = make(map[interface{}]interface{})
	err = session.Save(req, res)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if bUserName == "" {
		log.Println("inSituLogout: Empty session destroyed.")
	} else {
		log.Printf("inSituLogout: User %s logged out\n", bUserName)
	}
}

// testLoginStatus tests if a brucheion user is logged in.
//It takes the name of the function to build an appropriate message and the session to extract the user from
//and returns the name of the user, the message according to the test, and a boolean representing the login status.
func testLoginStatus(function string, session *sessions.Session) (user, message string, loggedin bool) {
	loggedin = false                       // before proven to be logged in, the login state should always be false
	if session.Values["Loggedin"] != nil { //test if the Loggedin variable has already been set
		if session.Values["Loggedin"].(bool) { //Session value "Loggedin" is true if user is already logged in
			var ok bool                                             //necessary so that program-wide variable user is changed instead of a new function variable is created.
			user, ok = session.Values["BrucheionUserName"].(string) //if session was valid get a username
			if !ok {                                                //error handling
				fmt.Println("func testLoginStatus: Type assertion failed.")
			}
			message = "func " + function + ": User " + user + " is logged in." //build appropriate message
			loggedin = true                                                    //set loggedin to true
		} else {
			message = "func " + function + ": \"Loggedin\" was false. User was not logged in." //build appropriate message
			loggedin = false                                                                   //set loggedin to false
		}
	} else {
		message = "func " + function + " \"Loggedin\" was nil. Session was not initialzed." //build appropriate message
		loggedin = false                                                                    //set loggedin to true
	}
	return user, message, loggedin //return username, message, and login state
}

// getSessionUser retrieves the Brucheion user name from a HTTP request. If it
// was properly validated with requireSession, the request session should
// be available in the request context. If not, the function will throw an
// error.
func getSessionUser(r *http.Request) (user string, err error) {
	session, ok := r.Context().Value("session").(*sessions.Session)
	if !ok {
		return "", errors.New("could not retrieve request session")
	}

	user, ok = session.Values["BrucheionUserName"].(string)
	if !ok {
		return "", errors.New("could not retrieve Brucheion user name from session")
	}
	return user, nil
}
