package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/boltdb/bolt"
)

//validateUserName guarantees that a user name was entered (not left blank)
//and that only numbers and letters were used.
func validateUserName(username string) *Validation {

	unameValidation := &Validation{ //create a validation object by reference
		ErrorCode: false}

	matched, err := regexp.MatchString("^[0-9a-zA-Z]*$", username) //create a regexp object to hold the regular expression to be tested
	if err != nil {
		fmt.Println("Wrong regex pattern.")
	}

	log.Println("func validateUserName: Validating: " + username)
	if strings.TrimSpace(username) == "" { //form was left blank
		log.Println("Username was left blank")
		unameValidation.Message = "Please enter a username." //the message will be displayed on the login page
		return unameValidation
	} else if !matched { //illegal characters were used
		log.Println("Username contained illegal characters")
		unameValidation.Message = "Please only use letters and numbers."
		return unameValidation
	} else { //a username only made of numbers and letters was used
		log.Println("Username valid")
		unameValidation.ErrorCode = true //the username was successfully validated
		return unameValidation
	}
}

//validateNoAuthUser checks whether the Brucheion username is already in use and
//whether it is already associated with a provider login.
func validateNoAuthUser(req *http.Request) (*Validation, error) {
	//prepares the noAuthUserValidation
	bUserValidation := &Validation{
		Message:       "An internal error occured. (This should never happen.)",
		ErrorCode:     false,
		BUserInUse:    false,
		BPAssociation: false}

	//get the session to retrieve session/cookie values
	session, err := getSession(req)
	if err != nil {
		return nil, err
	}

	//get user data from session
	brucheionUserName, ok := session.Values["BrucheionUserName"].(string)
	if !ok {
		fmt.Errorf("func validateNoAuthUser: Type assertion of brucheionUserName cookie value to string failed or session value could not be retrieved")
	}

	//open the user database
	userDB, err := openBoltDB(config.UserDB)
	if err != nil {
		return nil, err
	}
	defer userDB.Close()

	userDB.View(func(tx *bolt.Tx) error {

		//check if Username was used for Providerlogin (if a ProviderUser with that name exists)
		bucket := tx.Bucket([]byte("GitHub")) //open the GitHub Bucket and test whether there was an entry by this name. There should not be one though. (This is performed to ensure database integrity.)
		cursor := bucket.Cursor()
		for userID, PUserName := cursor.First(); userID != nil; userID, PUserName = cursor.Next() { //go through the users bucket and check for BUserName already in use
			if string(PUserName) == brucheionUserName { //if this Username is associated with an ID in the provider bucket than the database is corrupt in some way..
				bUserValidation.ErrorCode = false    //Error encountered (PUser in use, but not for this BUser)
				bUserValidation.BPAssociation = true //brucheionUserName was found in a provider bucket
			}
		}

		bucket = tx.Bucket([]byte("GitLab")) //open the GitLab Bucket and test whether there was an entry by this name. There should not be one though. (This is performed to ensure database integrity.)
		cursor = bucket.Cursor()
		for userID, PUserName := cursor.First(); userID != nil; userID, PUserName = cursor.Next() { //go through the users bucket and check for BUserName already in use
			if string(PUserName) == brucheionUserName { //if this Username is associated with an ID in the provider bucket than the database is corrupt in some way..
				bUserValidation.ErrorCode = false    //Error encountered (PUser in use, but not for this BUser)
				bUserValidation.BPAssociation = true //brucheionUserName was found in a provider bucket
			}
		}

		bucket = tx.Bucket([]byte("users"))                                                  //open the user bucket
		cursor = bucket.Cursor()                                                             //create a cursor object that will be used to iterate over database entries
		for BUserName, _ := cursor.First(); BUserName != nil; BUserName, _ = cursor.Next() { //go through the users bucket and check for BUserName already in use
			if string(BUserName) == brucheionUserName { //if this username was found in the users Bucket
				//buffer := bucket.Get([]byte(brucheionUserName)) //get the brucheionUser as []byte buffer
				//err := json.Unmarshal(buffer, &brucheionUser)   //unmarshal the buffer and save the brucheionUser in its variable
				//if err != nil {
				//	fmt.Println("Func validateNoAuthUser: Error unmarshalling brucheionUser: ", err) //this should never happen
				//}
				//BUPointer = &brucheionUser //set the pointer to the brucheionuser
				bUserValidation.BUserInUse = true
			}
		}

		if bUserValidation.BUserInUse && bUserValidation.BPAssociation { //if the user was found in the user bucket and a provider bucket (!bUserValidation.ErrorCode) Scenario (1)
			//log.Println("Scenario (1)")
			log.Println("Username already in use with a provider login")
			bUserValidation.Message = "Username " + brucheionUserName + " is already in use with a provider login. Please choose a different username."
		} else if bUserValidation.BUserInUse && !bUserValidation.BPAssociation { //if the user was found in the user bucket but not in a provider bucket Scenario (2)
			//log.Println("Scenario (2)")
			bUserValidation.ErrorCode = true
			bUserValidation.Message = "NoAuth user " + brucheionUserName + " found. Logged in."
		} else if !bUserValidation.BUserInUse && !bUserValidation.BPAssociation { //if the user was not found in the user bucket and neither in a provider bucket Scenario (3)
			//log.Println("Scenario (3)")
			bUserValidation.ErrorCode = true
			bUserValidation.Message = "New noAuth user " + brucheionUserName + " created. Logged in."
		}

		return nil //close DB view without an error
	})
	return bUserValidation, nil
}

//validateUser checks whether the Brucheion username is already in use,
//whether that username is associated with the same provider login (as chosen at the login screen),
//and whether that provider login is already in use with another Brucheion user.
func validateUser(req *http.Request) (*Validation, error) {

	bUserValidation := &Validation{
		Message:      "An internal error occured. (This should never happen.)",
		ErrorCode:    false,
		BUserInUse:   false,
		SameProvider: false,
		PUserInUse:   false}

	//get the session to retrieve session/cookie values
	session, err := getSession(req)
	if err != nil {
		return nil, err
	}

	//get user data from session
	brucheionUserName, ok := session.Values["BrucheionUserName"].(string)
	if !ok {
		fmt.Errorf("func validateUser: Type assertion of brucheionUserName cookie value to string failed or session value could not be retrieved")
	}
	log.Println("Debug: brucheionUserName = " + brucheionUserName)
	provider, ok := session.Values["Provider"].(string)
	if !ok {
		fmt.Errorf("func validateUser: Type assertion of provider cookie value to string failed or session value could not be retrieved")
	}
	log.Println("Debug: Provider = " + provider)
	providerUserID, ok := session.Values["ProviderUserID"].(string)
	if !ok {
		fmt.Errorf("func validateUser: Type assertion of ProviderUserID cookie value to string failed or session value could not be retrieved")
	}
	log.Println("Debug: providerUserID = \"" + providerUserID + "\"")

	userDB, err := openBoltDB(config.UserDB)
	if err != nil {
		return nil, err
	}
	defer userDB.Close() //always remember to close the DB

	userDB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("users")) //try to open the user bucket

		var brucheionUser BrucheionUser //create a BrucheionUser variable
		BUPointer := new(BrucheionUser) //create a pointer that will later be used to check if BrucheionUser is still empty
		BUPointer = nil                 //and make sure that it is empty
		cursor := bucket.Cursor()       //create a cursor object that will be used to iterate over database entries

		for BUserName, _ := cursor.First(); BUserName != nil; BUserName, _ = cursor.Next() { //go through the users bucket and check for BUserName already in use
			if string(BUserName) == brucheionUserName { //if this username was found in the users Bucket
				buffer := bucket.Get([]byte(brucheionUserName)) //get the brucheionUser as []byte buffer
				err := json.Unmarshal(buffer, &brucheionUser)   //unmarshal the buffer and save the brucheionUser in its variable
				if err != nil {
					fmt.Println("Error unmarshalling brucheionUser: ", err)
				}
				log.Println("Found BrucheionUser " + brucheionUserName + " in usersDB.")
				BUPointer = &brucheionUser //set the pointer to the brucheionuser
			}
		}

		//check if ProviderUser was used for other BUser
		if BUPointer == nil { //Login scenarios (4), (5)
			bUserValidation.BUserInUse = false   //This BUsername was not in use yet
			bUserValidation.SameProvider = false //No BUser -> No provider chosen
			bucket = tx.Bucket([]byte(provider)) //open the provider Bucket
			cursor := bucket.Cursor()
			for userID, _ := cursor.First(); userID != nil; userID, _ = cursor.Next() { //go through the provider bucket and check for BUserName already in use

				//Login scenario (4)
				if string(userID) == providerUserID { //if this userID was in the Bucket
					bUserValidation.Message = "This " + provider + " account is already in use for authentificating another login."
					bUserValidation.ErrorCode = false //Error encountered (PUser in use, but not for this BUser)
					bUserValidation.PUserInUse = true //ProviderUser from session already in use
				}
			}
			cursor = nil //necessary?
			//Login scenario (5)
			//TO DO: test alternative if statement:
			//if bUserValidation.BUserInUse == false
			//&& bUserValidation.SameProvider == false
			//&& bUserValidation.PUserInUse == false{
			//or if statement obsolete?
			if bUserValidation.Message == "An internal error occured. (This should never happen.)" { //If userID was not found in DB, message will be unaltered.
				bUserValidation.Message = "User " + brucheionUserName + " created for login with provider " + provider + ". Login successfull."
				bUserValidation.ErrorCode = true   //New BUser and new PUser -> Creating a new user is not an error
				bUserValidation.PUserInUse = false //ProviderUser not in use yet (could be omitted)
			}
		} else { //Login Scenarios (1), (2), (3)
			bUserValidation.BUserInUse = true       //The BrucheionUser has a representation in users.DB
			if provider == brucheionUser.Provider { //Login Scenarios (1), (2)
				bUserValidation.SameProvider = true                 //Provider from session and BrucheionUser match
				if providerUserID == brucheionUser.ProviderUserID { //if there was a user bucket and the session values match the DB values; Login Scenarios (1)
					bUserValidation.Message = "User " + brucheionUserName + " logged in successfully."
					bUserValidation.ErrorCode = true  //No error encountereddia
					bUserValidation.PUserInUse = true //ProviderUser from session and BrucheionUser match
				} else { //brucheionUser.ProviderUserID != providerUserID; Login Scenario (2)
					if providerUserID == "" {
						bUserValidation.Message = "Username " + brucheionUserName + " found in userDB. Redirecting to provider authentification."
						bUserValidation.ErrorCode = true     //Error encountered
						bUserValidation.SameProvider = false //The BUser is in Use with another Provider
						bUserValidation.PUserInUse = false   //ProviderUser from session and BrucheionUser don't match
					} else {
						bUserValidation.Message = "Username " + brucheionUserName + " is already registered with another account. Choose another username."
						bUserValidation.ErrorCode = false  //Error encountered
						bUserValidation.PUserInUse = false //ProviderUser from session and BrucheionUser don't match
					}
				}
			} else { //brucheionUser.Provider != provider; Login Scenario (3)
				log.Println("providerUserID =\"" + providerUserID + "\".")
				bUserValidation.Message = "Username " + brucheionUserName + " already in use with another provider."
				bUserValidation.ErrorCode = false    //Error encountered
				bUserValidation.SameProvider = false //The BUser is in Use with another Provider
				bUserValidation.PUserInUse = false   //ProviderUser from session and BrucheionUser don't match
			}
		}

		return nil //close DB view without an error
	})

	return bUserValidation, nil
}
