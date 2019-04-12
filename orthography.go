package main

import (
  "fmt"
  "regexp"
  "os"
  "encoding/json"
)

type OrthNormConfig struct {
  ReplacementsToUse      []RegexReplacement `json:"replacements_to_use"`
  ReplacementsToIgnore   []RegexReplacement `json:"replacements_to_ignore"`
}

type RegexReplacement struct {
  Name            string `json:"name"`
  Pattern         string `json:"pattern"`
  Replacement     string `json:"replacement"`
}

// Very slightly modified version of LoadConfiguration(), to accomodate different struct
func loadOrthNormConfig(lang string) OrthNormConfig {
  file = config.OrthNormFiles[lang]
  var newConfig OrthNormConfig               //initialize config as OrthNormConfig
  configFile, openFileError := os.Open(file) //attempt to open file
  defer configFile.Close()                   //push closing on call list
  if openFileError != nil {                  //error handling
    fmt.Println("Open file error: " + openFileError.Error())
  }
  jsonParser := json.NewDecoder(configFile) //initialize jsonParser with configFile
  jsonParser.Decode(&newConfig)             //parse configFile to newConfig
  return newConfig
}

// Applies regular expression replacements listed in JSON file to text string.
func orthographyNormalization(res http.ResponseWriter, req *http.Request) string {

  //First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("orthographyNormalization", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

  vars := mux.Vars(req)
  passageIDs := strings.Split(vars["urns"], "+") // comma canonically used for "page,line" notation



  var currLang string
  var currOrthNormConfig OrthNormConfig
  var texturns, texts []string



  // MOVE THESE EVENTUALLY
  lang = vars["lang"]
  orthNormConfig = loadOrthNormConfig(lang)



  switch len(passageIDs){
    // cp. collection.go, newCollection() for similar switch on variable length {urns} argument
    // multiple URNs separated with ","
    // here using SplitCTS rather than SplitCITE, since text rather than images

  case 0:
    // all passages in database
    // cp. cex.go, ExportCEX() for acting upon all buckets in db


      dbname := user + ".db"
      buckets := Buckets(dbname)
    	db, err := openBoltDB(dbname) //open bolt DB using helper function
    	if err != nil {
    		fmt.Printf("Error opening userDB: %s", err)
    		http.Error(res, err.Error(), http.StatusInternalServerError)
    		return
    	}
    	defer db.Close()
    	for i := range buckets {
    		db.View(func(tx *bolt.Tx) error {
    			// Assume bucket exists and has keys
    			b := tx.Bucket([]byte(buckets[i]))
    			c := b.Cursor()
    			for k, v := c.First(); k != nil; k, v = c.Next() {

    				retrievedjson := BoltURN{} // SOON Passage{}
    				json.Unmarshal([]byte(v), &retrievedjson)

            ctsurn := retrievedjson.URN   // SOON PassageID; USE TO DETERMINE LANGUAGE
    				text := retrievedjson.Text    // SOON Text.Brucheion; PERFORM TRANSFORMATION ON THIS

    				texturns = append(texturns, ctsurn) // NECESSARY?
    				texts = append(texts, text)         // NECESSARY?

            retrievedjson.Text.Normalised = ApplyChanges(text, orthNormConfig)   // ISN'T THIS THE POINT?

    			}

    			return nil
    		})
    	}














  case 1:
    // one passage
    // cp. again collection.go, newCollection() — or any number of other single-URN endpoints

    urn := gocite.SplitCTS(passageIDs[0])
    switch {
			case urn.InValid:
        io.WriteString(res, "failed")
        return
			default:
        // DO STUFF HERE
    }

  default:
    // specific multiple passages

    urn := gocite.SplitCTS(imageIDs[i])
		switch {
		case urn.InValid:
			continue
		default:
			// DO STUFF HERE
		}
  }
}

func ApplyChanges(text string, orthNormConfig OrthNormConfig) string {
  var replacements []RegexReplacement
  replacements = append(replacements, orthNormConfig.ReplacementsToUse...)
  for i := range replacements {
    re := regexp.MustCompile(replacements[i].Pattern)
  	text = re.ReplaceAllString(text, replacements[i].Replacement)
  }
  return text
}
