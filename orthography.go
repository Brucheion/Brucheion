package main

import (
  "fmt"
  "regexp"
  "os"
  "encoding/json"
  "strings"
  "log"
  "net/http"
  "github.com/ThomasK81/gocite"
  "github.com/gorilla/mux"
  "github.com/boltdb/bolt"
  "io"


)

type OrthographyNormalisationConfig struct {
  ReplacementsToUse      []RegexReplacement `json:"replacements_to_use"`
  ReplacementsToIgnore   []RegexReplacement `json:"replacements_to_ignore"`
}

type RegexReplacement struct {
  Name            string `json:"name"`
  Pattern         string `json:"pattern"`
  Replacement     string `json:"replacement"`
}

// results are pairs, with the latter ultimately to be stored in Normalised field of Passage.Text object (type EncText)
type NormalizationResult struct {
  PassageURN      string `json:"passageURN"`
  NormalizedText  string `json:"normalizedText"`
}

// cp. image.go, where items are single strings
type ResultJSONlist struct {
	Items []NormalizationResult `json:"items"`
}

// modified version of initialization.go, loadConfiguration()
func loadOrthographyNormalisationConfig(language_code string) OrthographyNormalisationConfig {
  configFilename := config.OrthographyNormalisationFilenames[language_code]
  var newConfig OrthographyNormalisationConfig  //initialize config as OrthographyNormalisationConfig
  configFile, openFileError := os.Open(configFilename) //attempt to open file
  defer configFile.Close()                   //push closing on call list
  if openFileError != nil {                  //error handling
    fmt.Println("Open file error: " + openFileError.Error())
  }
  jsonParser := json.NewDecoder(configFile) //initialize jsonParser with configFile
  jsonParser.Decode(&newConfig)             //parse configFile to newConfig
  return newConfig
}

// might be nice to add to work.go
func GetWorkLangFromCatalog(work_urn, dbname string) string {

  // adjust urn slightly to serve as bucket_id
  work_bucket_id := work_urn + ":"

  // fetch language_code from work bucket, using bucket_id as key to specify catalog data
  retrieved_catalog_value_data := BoltRetrieve(dbname, work_bucket_id, work_bucket_id)
  retrieved_cat_json := BoltCatalog{}
  json.Unmarshal([]byte(retrieved_catalog_value_data.JSON), &retrieved_cat_json)
  work_language_code := retrieved_cat_json.Language

  return work_language_code
}

func GetWorkURNFromPassageURN(passage_urn string) string {
  passage_urn_object := gocite.SplitCTS(passage_urn)
  work_urn := strings.Join([]string{
    passage_urn_object.Base,
    passage_urn_object.Protocol,
    passage_urn_object.Namespace,
    passage_urn_object.Work},
    ":")
  return work_urn
}

// needed since gocite func GetPassageByID(id string, w Work) requires Work object as input
// cp. above hypothetical funcs GetWorkByID(), GetAllWorks()
// might constitute basis for new file passage.go
// e.g. as GetPassageByID(id string) Passage
func GetPassageByURNOnly(passage_urn, dbname string) gocite.Passage {

  // derive work bucket id from passage_urn
  work_urn := GetWorkURNFromPassageURN(passage_urn)
  work_bucket_id := work_urn + ":"

  // fetch Passage object
  retrieved_passage_data := BoltRetrieve(dbname, work_bucket_id, passage_urn)
	passage_object := gocite.Passage{}
	json.Unmarshal([]byte(retrieved_passage_data.JSON), &passage_object)

  return passage_object
}

type WorkList struct {
  Works []gocite.Work
}


// might be nice to add to work.go
func GetAllWorks(dbname string) WorkList {
  var all_works = WorkList{}
  // to implement later...
  return all_works
}

// might be nice to add to work.go

// func GetWorkByID(id string) Work {
//   // to implement later...
// }

func PerformReplacements(text string, orthNormConfig OrthographyNormalisationConfig) string {
  var replacements []RegexReplacement
  replacements = append(replacements, orthNormConfig.ReplacementsToUse...)
  for i := range replacements {
    re := regexp.MustCompile(replacements[i].Pattern)
  	text = re.ReplaceAllString(text, replacements[i].Replacement)
  }
  return text
}

func normalizeOrthographyTemporarily(res http.ResponseWriter, req *http.Request) {

  //First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("normalizeOrthographyTemporarily", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

  // construct dbname
  dbname := user + ".db"

  // extract passage urn(s)
  vars := mux.Vars(req)
  passage_urns := strings.Split(vars["urns"], "+") // cannot use comma since canonically used for "page,line" notation
  // example urn arguments
  // "urn:cts:sktlit:skt0001.nyaya006.edYE:108,6/"
  // "urn:cts:sktlit:skt0001.nyaya006.edYE:108,6+urn:cts:sktlit:skt0001.nyaya006.edYE:108,10+urn:cts:sktlit:skt0001.nyaya006.edYE:108,20/"
  // or "all" to normalize everything in db

  // special argument for normalizing whole database
  if len(passage_urns) == 1 && passage_urns[0] == "all" {
    // redo passage_urns as slice with all passage urns for all works in whole database
    passage_urns = nil
    var all_works = GetAllWorks(dbname)
    for i := range all_works.Works {
        work_passages := all_works.Works[i].Passages
        for j := range work_passages {
          passage_urns = append(passage_urns, work_passages[j].PassageID)
        }
    }
  }

  // now process single or multiple specific passages

  // cp. similar in image.go
  response := ResultJSONlist{}
  var normalized_text_result string

  for i := range passage_urns {

    passage_urn := passage_urns[i]

    // derive work_urn from passage_urn
    work_urn := GetWorkURNFromPassageURN(passage_urn)

    // use work_urn to pick out appropriate orthography config replacements
    work_language_code := GetWorkLangFromCatalog(work_urn, dbname)
    orthographyNormalisationConfig := loadOrthographyNormalisationConfig(work_language_code)

    // fetch passage text
    passage := GetPassageByURNOnly(passage_urn, dbname)
    passage_text := passage.Text.Brucheion

    // normalize string
    normalized_text_result = PerformReplacements(passage_text, orthographyNormalisationConfig)

    // package passage_urn and normalized string as result
    response.Items = append(response.Items, NormalizationResult{passage_urn, normalized_text_result})

  }

  resultJSON, _ := json.Marshal(response)
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(res, string(resultJSON))

}



// close cousin of normalizeOrthographyTemporarily
// I don't yet understand how to properly receive a JSON response from an endpoint for further processing
// therefore, in order to keep developing this part, this alternative endpoint simply saves along the way
func normalizeOrthographyAndSave(res http.ResponseWriter, req *http.Request) {

  //First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("normalizeOrthographyTemporarily", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

  // construct dbname
  dbname := user + ".db"

  // extract passage urn(s)
  vars := mux.Vars(req)
  passage_urns := strings.Split(vars["urns"], "+") // cannot use comma since canonically used for "page,line" notation
  // example urn arguments
  // "urn:cts:sktlit:skt0001.nyaya006.edYE:108,6/"
  // "urn:cts:sktlit:skt0001.nyaya006.edYE:108,6+urn:cts:sktlit:skt0001.nyaya006.edYE:108,10+urn:cts:sktlit:skt0001.nyaya006.edYE:108,20/"
  // or "all" to normalize everything in db

  // special argument for normalizing whole database
  if len(passage_urns) == 1 && passage_urns[0] == "all" {
    // redo passage_urns as slice with all passage urns for all works in whole database
    passage_urns = nil
    var all_works = GetAllWorks(dbname)
    for i := range all_works.Works {
        work_passages := all_works.Works[i].Passages
        for j := range work_passages {
          passage_urns = append(passage_urns, work_passages[j].PassageID)
        }
    }
  }

  // now process single or multiple specific passages

  // cp. similar in image.go
  var normalized_text_result string

  for i := range passage_urns {

    passage_urn := passage_urns[i]

    // derive work_urn from passage_urn
    work_urn := GetWorkURNFromPassageURN(passage_urn)

    // use work_urn to pick out appropriate orthography config replacements
    work_language_code := GetWorkLangFromCatalog(work_urn, dbname)
    orthographyNormalisationConfig := loadOrthographyNormalisationConfig(work_language_code)

    // fetch passage text
    passage := GetPassageByURNOnly(passage_urn, dbname)
    passage_text := passage.Text.Brucheion

    // normalize string
    normalized_text_result = PerformReplacements(passage_text, orthographyNormalisationConfig)

    // DIFFERENT BELOW HERE

    // update temporary object with result
    passage.Text.Normalised = normalized_text_result

    fmt.Println("passage.Text.Normalised: ", passage.Text.Normalised)

    // save updated object to database
    updatednode, _ := json.Marshal(passage)
    db, err := openBoltDB(dbname)
    if err != nil {
      fmt.Printf("Error opening userDB: %s", err)
      http.Error(res, err.Error(), http.StatusInternalServerError)
      return
    }
    fmt.Println("HERE 1")
    key := []byte(passage.PassageID)    //
    value := []byte(updatednode) //
    // store some data
    err = db.Update(func(tx *bolt.Tx) error {
      fmt.Println("HERE 3")
      bucket, err := tx.CreateBucketIfNotExists([]byte(work_urn+":"))
      fmt.Println("HERE 4")
      if err != nil {
        return err
      }
      fmt.Println("HERE 5")
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
    db.Close()
    fmt.Println("HERE 6")
  }

  io.WriteString(res, "normalization successfully saved")
}
