package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/boltdb/bolt"

	"github.com/gorilla/sessions" //for Cookiestore and other session functionality
	"github.com/markbates/goth/gothic"
)

var BrucheionStore sessions.Store

//The sessionName of the Brucheion Session
const SessionName = "brucheionSession"

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

	if *noAuth {
		log.Println("noAuth flag was true")
	} else {
		log.Println("noAuth flag was false")
	}

	lp := &LoginPage{
		Title:  "Brucheion Login Page",
		NoAuth: *noAuth}
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
	} else { //Destroy the session we just got if user was not logged in yet (proceed with login process)
		session.Options.MaxAge = -1
		session.Values = make(map[interface{}]interface{})
		err = session.Save(req, res)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	title := "Brucheion Login Page" //set the title of the page

	//populates Loginpage with basic data and the form values
	lp := &LoginPage{
		BUserName: strings.TrimSpace(req.FormValue("brucheionusername")),
		Host:      config.Host,
		Title:     title}

	unameValidation := ValidateUserName(lp.BUserName) //checks if this username only has (latin) letters and (arabian) numbers

	if unameValidation.ErrorCode { //if a valid username has been chosen
		session, err = InitializeSession(req) //initialize a persisting session
		if err != nil {
			fmt.Println("LoginPOST: Error initializing the session.")
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		//save the BrucheionUserName in the session as well and set the Loggedin value to false
		session.Values["BrucheionUserName"] = lp.BUserName
		session.Values["Loggedin"] = false
		session.Save(req, res)

		err = InitializeUserDB() //Make sure the userDB file is there and has the necessary buckets.
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		if *noAuth { //if the noauth flag was set true: check if username is not in use for a login with a provider
			log.Println("noAuth flag was true")
			lp.NoAuth = *noAuth

			validation, err := ValidateNoAuthUser(req) //validate if credentials match existing user and not in use with a provider login yet
			if err != nil {
				fmt.Printf("\nLoginPOST error validating user: %s", err)
				http.Error(res, err.Error(), http.StatusInternalServerError)
			}
			if validation.ErrorCode {
				brucheionUser := &BrucheionUser{ //create Brucheionuser instance
					BUserName: lp.BUserName,
					Provider:  "noAuth"}
				if !validation.BUserInUse { // create new noAuth user if the username was not in use
					db, err := OpenBoltDB(config.UserDB) //open bolt DB using helper function
					if err != nil {
						fmt.Printf("Error opening userDB: %s", err)
						http.Error(res, err.Error(), http.StatusInternalServerError)
						return
					}
					db.Update(func(tx *bolt.Tx) error {
						bucket := tx.Bucket([]byte("users"))
						buffer, err := json.Marshal(brucheionUser) //Marshal user data
						if err != nil {
							fmt.Errorf("Failed marshalling user data for user %s: %s\n", brucheionUser.BUserName, err)
							return err
						}
						err = bucket.Put([]byte(brucheionUser.BUserName), buffer) //put user into bucket
						if err != nil {
							fmt.Errorf("Failed saving user %s in users.db\n", brucheionUser.BUserName, err)
							return err
						}
						fmt.Printf("Successfully saved new user %s in users.DB.\n", brucheionUser.BUserName)

						fmt.Println(validation.Message) //Display validation.Message if all went well.
						return nil
					})
					db.Close() //always remember to close the db
				}
				session.Values["Loggedin"] = true //To keep the user logged in
				session.Save(req, res)
				lp.Message = validation.Message //The message to be replied in regard to the login scenario
				renderAuthTemplate(res, "callback", lp)
			} else {
				lp.Message = validation.Message       //add the message to the loginpage
				renderLoginTemplate(res, "login", lp) //and render the login template again, displaying said message.
			}
		} else { //if the noauth flag was not set, or set false: continue with authentification using a provider
			lp.Provider = req.FormValue("provider")
			authPath := "/auth/" + strings.ToLower(lp.Provider) + "/" //set up the path for redirect according to provider
			session.Values["Provider"] = lp.Provider                  //the provider used for login
			session.Save(req, res)                                    //always save the session after setting values
			http.Redirect(res, req, authPath, http.StatusFound)       //redirect to auth page with correct provider
		}
	} else { //if the the user name was not valid
		lp.Message = unameValidation.Message  //add the message to the loginpage
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
		} else { //proceed with login process (gothic redirects to provider and redirects to callback)
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
			//create new entry for new BUser
			db, err := OpenBoltDB(config.UserDB) //open bolt DB using helper function
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

//Logout kills the session (equivalent to logging out) and logs the logout.
func Logout(res http.ResponseWriter, req *http.Request) {

	session, err := GetSession(req)
	if err != nil {
		fmt.Errorf("No session, no logout")
		return
	}

	bUserName, ok := session.Values["BrucheionUserName"].(string)
	if !ok {
		fmt.Println("func Logout: Type assertion of value BrucheionUserName to string failed or session value could not be retrieved.")
	}

	session.Options.MaxAge = -1
	session.Values = make(map[interface{}]interface{})
	err = session.Save(req, res)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("User %s has logged out\n", bUserName)
	http.Redirect(res, req, "/login/", http.StatusFound)

}

//TestLoginStatus returns the tests if a user is logged in.
//It takes the name of the function to build an appropriate message and the session to extract the user from
//and returns the name of the user, the message according to the test, and a boolean representing the login status.
func TestLoginStatus(function string, session *sessions.Session) (user string, message string, loggedin bool) {
	loggedin = false                       // before proven to be logged in, the login state should always be false
	if session.Values["Loggedin"] != nil { //test if the Loggedin variable has already been set
		if session.Values["Loggedin"].(bool) { //"Loggedin" will be true if user is already logged in
			ok := false                                             //necessary so that fuction-wide variable user is changed instead of a new variable being created.
			user, ok = session.Values["BrucheionUserName"].(string) //if session was valid get a username
			if !ok {                                                //error handling
				fmt.Println("func TestLoginStatus: Type assertion to string failed for session value BrucheionUser or session value could not be retrieved.")
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
