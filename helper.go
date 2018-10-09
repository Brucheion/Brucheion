package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"

	//"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/ThomasK81/gocite"

	"github.com/markbates/goth" //for login using external providers
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/gitlab"

	"github.com/gorilla/securecookie" //for generating the cookieStore key
	"github.com/gorilla/sessions"     //for Cookiestore and other session functionality

	"github.com/boltdb/bolt"
)

// Sort-Matrix Interface

type dataframe struct {
	Indices []int
	Values1 []string
	Values2 []string
}

//var config Config

func (m dataframe) Len() int           { return len(m.Indices) }
func (m dataframe) Less(i, j int) bool { return m.Indices[i] < m.Indices[j] }
func (m dataframe) Swap(i, j int) {
	m.Indices[i], m.Indices[j] = m.Indices[j], m.Indices[i]
	m.Values1[i], m.Values1[j] = m.Values1[j], m.Values1[i]
	m.Values2[i], m.Values2[j] = m.Values2[j], m.Values2[i]
}

func getContent(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Status error: %v", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Read body: %v", err)
	}

	return data, nil
}

func removeDuplicatesUnordered(elements []string) []string {
	encountered := map[string]bool{}

	// Create a map of all unique elements.
	for v := range elements {
		encountered[elements[v]] = true
	}

	// Place all keys from the map into a slice.
	result := []string{}
	for key := range encountered {
		result = append(result, key)
	}
	return result
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Helper function to pull the href attribute from a Token
func getHref(t html.Token) (ok bool, href string) {
	for _, a := range t.Attr {
		if a.Key == "href" {
			href = a.Val
			ok = true
		}
	}
	return
}

func extractLinks(urn gocite.Cite2Urn) (links []string, err error) {
	urnLink := urn.Namespace + "/" + strings.Replace(urn.Collection, ".", "/", -1) + "/"
	url := config.Host + "/static/image_archive/" + urnLink
	response, err := http.Get(url)
	if err != nil {
		return links, err
	}
	z := html.NewTokenizer(response.Body)
	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return
		case tt == html.StartTagToken:
			t := z.Token()

			isAnchor := t.Data == "a"
			if !isAnchor {
				continue
			}
			ok, url := getHref(t)
			if strings.Contains(url, ".dzi") {
				urnStr := urn.Base + ":" + urn.Protocol + ":" + urn.Namespace + ":" + urn.Collection + ":" + strings.Replace(url, ".dzi", "", -1)
				links = append(links, urnStr)
			}
			if !ok {
				continue
			}
		}
	}
	return links, nil
}

//SetUpGothic sets up Gothic for login procedure
func SetUpGothic() {
	//Build the authentification paths for the choosen providers
	gitHubPath := (config.Host + "/auth/github/callback")
	gitLabPath := (config.Host + "/auth/gitlab/callback")
	//Tell gothic which login providers to use
	goth.UseProviders(
		github.New(config.GitHubKey, config.GitHubSecret, gitHubPath),
		gitlab.New(config.GitLabKey, config.GitLabSecret, gitLabPath, config.GitLabScope))
	//Create new Cookiestore for _gothic_session
	loginTimeout := 60 //Time the _BrucheionSession cookie will be alive in seconds
	gothic.Store = GetCookieStore(loginTimeout)
}

//LoadConfiguration loads and parses the JSON config file and returns Config.
func LoadConfiguration(file string) Config {
	var newConfig Config                       //initialize config as Config
	configFile, openFileError := os.Open(file) //attempt to open file
	defer configFile.Close()                   //push closing on call list
	if openFileError != nil {                  //error handling
		fmt.Println("Open file error: " + openFileError.Error())
	}
	jsonParser := json.NewDecoder(configFile) //initialize jsonParser with configFile
	jsonParser.Decode(&newConfig)             //parse configFile to config
	return newConfig                          //return ServerConfig config
}

//GetCookieStore sets up and returns a cookiestore. The maxAge is defined by what was defined in config.json.
func GetCookieStore(maxAge int) sessions.Store {
	//Todo: research encryption key and if it can/should be used fot our use cases
	key := securecookie.GenerateRandomKey(64) //Generate a random key for the session
	if key == nil {
		fmt.Println("Error generating random session key.")
	}

	cookieStore := sessions.NewCookieStore([]byte(key)) //Get CookieStore from sessions package
	cookieStore.Options.HttpOnly = true                 //Ensures that Cookie can not be accessed by scripts
	cookieStore.MaxAge(maxAge)                          //Sets the maxAge of the session/cookie

	return cookieStore
}

//InitializeSession will create and return the session and set the session options
func InitializeSession(req *http.Request) (*sessions.Session, error) {
	fmt.Println("Initializing session for " + SessionName)
	session, err := BrucheionStore.Get(req, SessionName)
	if err != nil {
		fmt.Printf("GetSession: Error getting the session: %s\n", err)
		return nil, err
	}
	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   config.MaxAge,
		HttpOnly: true}
	return session, nil
}

//GetSession will return an open session when there is a matching session by that name, and valid for the request.
//Note that it will also return a new session, if none was open by that name. -> Close the session after testing.
func GetSession(req *http.Request) (*sessions.Session, error) {
	session, err := BrucheionStore.Get(req, SessionName)
	if err != nil {
		fmt.Printf("GetSession: Error getting the session: %s\n", err)
		return nil, err
	}
	return session, nil
}

//OpenBoltDB returns an opened Bolt Database for dbName.
func OpenBoltDB(dbName string) (*bolt.DB, error) {

	db, err := bolt.Open(dbName, 0600, &bolt.Options{Timeout: 30 * time.Second}) //open DB with - wr- --- ---
	if err != nil {
		return nil, err
	}
	//fmt.Println("DB opened")
	return db, nil
}

//initializeUserDB should be called once during login attempt to make sure that all buckets are in place.
func InitializeUserDB() error {
	fmt.Println("Initializing UserDB")
	db, err := OpenBoltDB(config.UserDB)
	if err != nil {
		return err
	}

	//create the three buckets needed: users, GitHub, GitLab
	db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("users")) //create a new bucket to store new user
		if err != nil {
			fmt.Errorf("Failed creating bucket users: %s", err)
			return err
		}
		bucket, err = tx.CreateBucketIfNotExists([]byte("GitHub"))
		if err != nil {
			fmt.Errorf("Failed creating bucket GitHub: %s", err)
			return err
		}
		bucket, err = tx.CreateBucketIfNotExists([]byte("GitLab"))
		if err != nil {
			fmt.Errorf("Failed creating bucket GitLab: %s", err)
			return err
		}
		_ = bucket //to have done something with the bucket (avoiding 'username declared and not used' error)

		return nil //if all went well, error can be returned with <nil>
	})

	db.Close() //always remember to close the db
	//fmt.Println("DB closed")
	return nil
}

func ValidateUserName(username string) *Validation {

	unameValidation := &Validation{
		ErrorCode: false}

	matched, err := regexp.MatchString("^[0-9a-zA-Z]*$", username)
	if err != nil {
		fmt.Println("Wrong regex pattern.")
	}

	fmt.Println("Validating: " + username)
	if strings.TrimSpace(username) == "" {
		unameValidation.Message = "Please enter a username."
		return unameValidation
	} else if !matched {
		unameValidation.Message = "Please only use letters and numbers."
		return unameValidation
	} else {
		unameValidation.ErrorCode = true
		return unameValidation
	}
}

func ValidateUser(req *http.Request) (*Validation, error) {
	bUserValidation := &Validation{
		Message:      "An internal error occured. (This should never happen.)",
		ErrorCode:    false,
		BUserInUse:   false,
		SameProvider: false,
		PUserInUse:   false}

	//get the session to retrieve session/cookie values
	session, err := GetSession(req)
	if err != nil {
		return nil, err
	}

	//get user data from session
	brucheionUserName, ok := session.Values["BrucheionUserName"].(string)
	if !ok {
		fmt.Errorf("Func ValidateUser: Type assertion of brucheionUserName cookie value to string failed or session value could not be retrieved.")
	}
	provider, ok := session.Values["Provider"].(string)
	if !ok {
		fmt.Errorf("Func ValidateUser: Type assertion of provider cookie value to string failed or session value could not be retrieved.")
	}
	/*providerNickName, ok := session.Values["ProviderNickName"].(string)
	if !ok {
		fmt.Errorf("Func ValidateUser: Type assertion of ProviderNickName cookie value to string failed or session value could not be retrieved.")
	}*/
	providerUserID, ok := session.Values["ProviderUserID"].(string)
	if !ok {
		fmt.Errorf("Func ValidateUser: Type assertion of ProviderUserID cookie value to string failed or session value could not be retrieved.")
	}

	/*fmt.Println("Debugging values from session:")
	fmt.Println("brucheionUserName: " + brucheionUserName)
	fmt.Println("provider: " + provider)
	fmt.Println("providerNickName: " + providerNickName)
	fmt.Println("providerUserID: " + providerUserID)

	fmt.Printf("\nValidateUser: Attempting to open: %s\n", config.UserDB)*/
	userDB, err := OpenBoltDB(config.UserDB)
	if err != nil {
		return nil, err
	}
	defer userDB.Close()

	userDB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("users")) //try to open the user bucket

		var brucheionUser BrucheionUser //create a pointer to a BrucheionUser variable
		pointer := new(BrucheionUser)
		cursor := bucket.Cursor()
		pointer = nil
		for BUserName, _ := cursor.First(); BUserName != nil; BUserName, _ = cursor.Next() { //go through the users bucket and check for BUserName already in use
			//Login scenario (4)
			if string(BUserName) == brucheionUserName { //if this userID was in the Bucket
				buffer := bucket.Get([]byte(brucheionUserName)) //get the brucheionUser as []byte buffer
				err := json.Unmarshal(buffer, &brucheionUser)   //unmarshal the buffer
				if err != nil {
					fmt.Println("Error unmarshalling brucheionUser: ", err)
				}
				pointer = &brucheionUser //set the pointer to the brucheionuser
				/*fmt.Println("&brucheionUser")
				fmt.Println(&brucheionUser)
				/*fmt.Println("*brucheionUser")
				fmt.Println(*brucheionUser)*/
			}
		}

		//check if ProviderUser was used for other BUser
		if pointer == nil { //Login scenarios (4), (5)
			bUserValidation.BUserInUse = false   //This BUsername was not in use yet
			bUserValidation.SameProvider = false //No BUser -> No provider chosen
			bucket = tx.Bucket([]byte(provider)) //open the provider Bucket
			cursor := bucket.Cursor()
			//for userID, nickName := cursor.First(); userID != nil; userID, nickName = cursor.Next() { //go through the users bucket and check for BUserName already in use
			for userID, _ := cursor.First(); userID != nil; userID, _ = cursor.Next() { //go through the users bucket and check for BUserName already in use

				//Login scenario (4)
				if string(userID) == providerUserID { //if this userID was in the Bucket
					bUserValidation.Message = "This " + provider + " account is already in use for authentificating another login."
					bUserValidation.ErrorCode = false //Error encoungtered (PUser in use, but not for this BUser)
					bUserValidation.PUserInUse = true //ProviderUser from session already in use
				}
			}
			cursor = nil
			//Login scenario (5)
			if bUserValidation.Message == "An internal error occured. (This should never happen.)" { //If userID was not found in DB, message will be unaltered.
				bUserValidation.Message = "User " + brucheionUserName + " created for login with provider " + provider + ". Login successfull."
				bUserValidation.ErrorCode = true   //New BUser and new PUser -> Creating a new user is not an error
				bUserValidation.PUserInUse = false //ProviderUser not in use yet
			}
		} else { //Login Scenarios (1), (2), (3)
			bUserValidation.BUserInUse = true       //The BrucheionUser has a representation in users.DB
			if provider == brucheionUser.Provider { //Login Scenarios (1), (2)
				bUserValidation.SameProvider = true                 //Provider from session and BrucheionUser match
				if providerUserID == brucheionUser.ProviderUserID { //if there was a user bucket and the session values match the DB values; Login Scenarios (1)
					bUserValidation.Message = "User " + brucheionUserName + " logged in successfully."
					bUserValidation.ErrorCode = true  //No error encountered
					bUserValidation.PUserInUse = true //ProviderUser from session and BrucheionUser match
				} else { //brucheionUser.ProviderUserID != providerUserID; Login Scenarios (2)
					bUserValidation.Message = "Username " + brucheionUserName + " is already registered with another account. Choose another username."
					bUserValidation.ErrorCode = false  //Error encountered
					bUserValidation.PUserInUse = false //ProviderUser from session and BrucheionUser don't match
				}
			} else { //brucheionUser.Provider != provider; Login Scenario (3)
				bUserValidation.Message = "Username " + brucheionUserName + " already in use with another provider."
				bUserValidation.ErrorCode = false    //Error encountered
				bUserValidation.SameProvider = false //The BUser is in Use with another Provider
				bUserValidation.PUserInUse = false   //ProviderUser from session and BrucheionUser don't match
			}
		}
		/*fmt.Println("Debugging bUser retrieved from DB:")
		fmt.Println("BUserName: " + brucheionUser.BUserName)
		fmt.Println("Provider: " + brucheionUser.Provider)
		fmt.Println("providerNickName: " + brucheionUser.PUserName)
		fmt.Println("ProviderUserID: " + brucheionUser.ProviderUserID)*/

		return nil //close DB view without an error
	})

	/*fmt.Println("Print Debugging db.Update: ")
	fmt.Println("validation.ErrorCode: " + strconv.FormatBool(bUserValidation.ErrorCode))
	fmt.Println("validation.BUserInUse: " + strconv.FormatBool(bUserValidation.BUserInUse))
	fmt.Println("validation.SameProvider: " + strconv.FormatBool(bUserValidation.SameProvider))
	fmt.Println("validation.PUserInUse: " + strconv.FormatBool(bUserValidation.PUserInUse))*/

	return bUserValidation, nil
}

func Logout(res http.ResponseWriter, req *http.Request) {

	session, err := GetSession(req)
	if err != nil {
		fmt.Errorf("No session, no logout")
		return
	}

	bUserName, ok := session.Values["BrucheionUserName"].(string)
	if !ok {
		fmt.Println("Func Logout: Type assertion of value BrucheionUserName to string failed or session value could not be retrieved.")
	}

	session.Options.MaxAge = -1
	session.Values = make(map[interface{}]interface{})
	err = session.Save(req, res)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("User %s has logged out\n", bUserName)
	http.Redirect(res, req, "/login/", http.StatusFound)

}
