package main

import (
  "encoding/json"
  "fmt"
  "github.com/ThomasK81/gocite"
  "github.com/boltdb/bolt"
  "github.com/gorilla/mux"
  "io"
  "log"
  "net/http"
  "os"
  "regexp"
  "strings"
)

type OrthographyNormalisationConfig struct {
  ReplacementsToUse    []RegexReplacement `json:"replacements_to_use"`
  ReplacementsToIgnore []RegexReplacement `json:"replacements_to_ignore"`
}

type RegexReplacement struct {
  Name        string `json:"name"`
  Pattern     string `json:"pattern"`
  Replacement string `json:"replacement"`
}

// results are pairs, with the latter ultimately to be stored in Normalised field of Passage.Text object (type EncText)
type NormalizationResult struct {
  PassageURN     string `json:"passageURN"`
  NormalizedText string `json:"normalizedText"`
}

// cp. image.go, where items are single strings
type ResultJSONlist struct {
  Items []NormalizationResult `json:"items"`
}

// modified version of initialization.go, loadConfiguration()
func loadOrthographyNormalisationConfig(language_code string) OrthographyNormalisationConfig {
  configFilename := config.OrthographyNormalisationFilenames[language_code]
  var newConfig OrthographyNormalisationConfig         //initialize config as OrthographyNormalisationConfig
  configFile, openFileError := os.Open(configFilename) //attempt to open file
  defer configFile.Close()                             //push closing on call list
  if openFileError != nil {                            //error handling
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
  retrieved_catalog_value_data, _ := BoltRetrieve(dbname, work_bucket_id, work_bucket_id)
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
  retrieved_passage_data, _ := BoltRetrieve(dbname, work_bucket_id, passage_urn)
  passage_object := gocite.Passage{}
  json.Unmarshal([]byte(retrieved_passage_data.JSON), &passage_object)

  return passage_object
}

// might be nice to add to work.go
type WorkList struct {
  Works []gocite.Work
}

func GetAllWorks(dbname string) WorkList {
  var all_works = WorkList{}
  // to implement later...
  return all_works
}

// might be nice to add to a "passage.go"
type PassageList struct {
  Items []gocite.Passage
}

func GetAllPassages(dbname string) PassageList {
  passage_list := PassageList{}
  buckets := Buckets(dbname)
  db, err := openBoltDB(dbname) //open bolt DB using helper function
  if err != nil {
    fmt.Printf("Error opening userDB: %s", err)
    return passage_list
  }
  defer db.Close()
  for i := range buckets {
    db.View(func(tx *bolt.Tx) error {
      // Assume bucket exists and has keys
      b := tx.Bucket([]byte(buckets[i]))
      c := b.Cursor()
      for k, v := c.First(); k != nil; k, v = c.Next() {
        retrieved_passage := gocite.Passage{}
        json.Unmarshal([]byte(v), &retrieved_passage)
        if retrieved_passage.Text.Brucheion != "" {
          passage_list.Items = append(passage_list.Items, retrieved_passage)
        }
      }
      return nil
    })
  }
  return passage_list
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
    var passage_list = GetAllPassages(dbname)
    for i := range passage_list.Items {
      passage_urns = append(passage_urns, passage_list.Items[i].PassageID)
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
// I don't yet understand how to properly receive a JSON response back from an endpoint for further processing
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
    var passage_list = GetAllPassages(dbname)
    for i := range passage_list.Items {
      passage_urns = append(passage_urns, passage_list.Items[i].PassageID)
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

    // save updated object to database
    updatednode, _ := json.Marshal(passage)
    db, err := openBoltDB(dbname)
    if err != nil {
      fmt.Printf("Error opening userDB: %s", err)
      http.Error(res, err.Error(), http.StatusInternalServerError)
      return
    }
    key := []byte(passage.PassageID) //
    value := []byte(updatednode)     //
    // store some data
    err = db.Update(func(tx *bolt.Tx) error {
      bucket, err := tx.CreateBucketIfNotExists([]byte(work_urn + ":"))
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
    db.Close()
  }

  io.WriteString(res, "normalization successfully saved")
}
