package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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

//getContent returns the response data from a GET request using url from parameter as a byte slice.
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

//contains returns true if the needle string is found in the heystack string slice
func contains(heystack []string, needle string) bool {
	for _, straw := range heystack {
		if straw == needle {
			return true
		}
	}
	return false
}

//getHref pulls the href attribute from a html Token
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

//maxfloat returns the index of the highest float64 in a float64 slice
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

//testString does not seem to be in use anymore (?)
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

//testStringSL is used for multipage alignment (?)
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

//testAllTheSame tests if all strings in a two-dimensional string-slice are the same
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

//findSpace returns a rune-slice exluding leading and trailing whitespaces
//(Works like strings.TrimSpace but for rune-slices)
//Additionally returns the count of trimmed leading and trailing whitespaces
//Used in addSansHyphens
func findSpace(runeSl []rune) (spBefore, spAfter int, newSl []rune) {
	spAfter = 0
	spBefore = 0
	for i := 0; i < len(runeSl); i++ {
		if runeSl[i] == rune(' ') {
			spBefore++ //continue
		} else {
			//since i is incremented anyway, wouldn't it be enough to set spBefore = i-1 here?
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

//addSansHyphens adds Hyphens after certain sanscrit runes but not before
//used for nwa and multipage alignment
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

//SetUpGothic sets up Gothic for login procedure
func setUpGothic() {
	//Build the authentification paths for the choosen providers
	gitHubPath := (config.Host + "/auth/github/callback")
	gitLabPath := (config.Host + "/auth/gitlab/callback")
	//Tell gothic which login providers to use
	goth.UseProviders(
		github.New(config.GitHubKey, config.GitHubSecret, gitHubPath),
		gitlab.New(config.GitLabKey, config.GitLabSecret, gitLabPath, config.GitLabScope))
	//Create new Cookiestore for _gothic_session
	loginTimeout := 60 //Time the _gothic_session cookie will be alive in seconds
	gothic.Store = getCookieStore(loginTimeout)
}

//LoadConfiguration loads and parses the JSON config file and returns Config.
func loadConfiguration(file string) Config {
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

//getCookieStore sets up and returns a cookiestore. The maxAge is defined by what was defined in config.json.
func getCookieStore(maxAge int) sessions.Store {
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

//initializeSession will create and return the session and set the session options
func initializeSession(req *http.Request) (*sessions.Session, error) {
	log.Println("Initializing session: " + SessionName)
	session, err := BrucheionStore.Get(req, SessionName)
	if err != nil {
		fmt.Printf("InitializeSession: Error getting the session: %s\n", err)
		return nil, err
	}
	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   config.MaxAge,
		HttpOnly: true}
	return session, nil
}

//getSession will return an open session when there is a matching session by that name, and valid for the request.
//Note that it will also return a new session, if none was open by that name. -> Close the session after testing.
func getSession(req *http.Request) (*sessions.Session, error) {
	session, err := BrucheionStore.Get(req, SessionName)
	if err != nil {
		fmt.Printf("getSession: Error getting the session: %s\n", err)
		return nil, err
	}
	return session, nil
}

//openBoltDB returns an opened Bolt Database for given dbName.
func openBoltDB(dbName string) (*bolt.DB, error) {
	db, err := bolt.Open(dbName, 0600, &bolt.Options{Timeout: 30 * time.Second}) //open DB with - wr- --- ---
	if err != nil {
		return nil, err
	}
	//fmt.Println("DB opened")
	return db, nil
}

//initializeUserDB should be called once during login attempt to make sure that all buckets are in place.
func initializeUserDB() error {
	log.Println("Initializing UserDB")
	db, err := openBoltDB(config.UserDB)
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
