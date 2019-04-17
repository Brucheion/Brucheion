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

// might be nice to add to work.go

// func GetAllWorks(dbname string) []gocite.Work {
//   // to implement later...
// }

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

func normalizeOrthography(res http.ResponseWriter, req *http.Request) {

  //First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("normalizeOrthography", session)
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

  // cp. similar in image.go
  response := ResultJSONlist{}
  var normalized_text_result string

  switch {
  case len(passage_urns) == 1 && passage_urns[0] == "all": // all passages in all works in database

    // not implemented yet...
    // part of this will be implementing work.go, GetAllWorks() (see above)
    // then will loop over Works, build up cumulative all_Passages from Work field Passages ([]Passage)
    // then simply process each Passage in same way as below, end by returning giant JSON response

    fmt.Println("this will eventually normalize all passages... ")

  default: // single or multiple specific passages

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

  }

  resultJSON, _ := json.Marshal(response)
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(res, string(resultJSON))

}
