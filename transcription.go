package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ThomasK81/gocite"

	"github.com/gorilla/mux"

	"github.com/boltdb/bolt"
)

// SaveTranscription parses a transcription from the http.Request
//and saves it to the corresponding URN bucket in the user database
func SaveTranscription(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("SaveTranscription", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	newkey := vars["key"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	text := req.FormValue("text")
	linetext := text
	//linetext := strings.Split(text, "\r\n")
	text = strings.Replace(text, "\r\n", "", -1)
	dbname := user + ".db"
	retrieveddata, _ := BoltRetrieve(dbname, newbucket, newkey)
	retrievedjson := gocite.Passage{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	retrievedjson.Text.Brucheion = text //gocite.Passage.Text.Brucheion is the text representation with newline tags
	retrievedjson.Text.TXT = linetext   //gocite.Passage.Text.TXT is the text representation with real line breaks instead of newline tags
	newnode, _ := json.Marshal(retrievedjson)
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		log.Println(fmt.Printf("SaveTranscription: error opening userDB: %s", err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
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
	http.Redirect(res, req, "/view/"+newkey, http.StatusFound)
}
