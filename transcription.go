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

func SaveTranscriptionGET(res http.ResponseWriter, req *http.Request) {
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
	log.Println(user)
	log.Println(req.Method)
}

// SaveTranscription parses a transcription from the http.Request
//and saves it to the corresponding URN bucket in the user database
func SaveTranscriptionPOST(res http.ResponseWriter, req *http.Request) {

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
	log.Println(req.Method)
	/*if r.Method != "POST" {
		io.WriteString(w, "Only POST is supported!")
		return
	}*/
	log.Println(req.ParseForm())
	log.Println(req.FormValue("text"))
	text := req.FormValue("text")
	log.Println("Debug: text=" + text)
	test := req.FormValue("test")
	log.Println("Debug: test=" + test)
	linetext := text
	//linetext := strings.Split(text, "\r\n")
	text = strings.Replace(text, "\r\n", "", -1)
	dbname := user + ".db"
	retrieveddata := BoltRetrieve(dbname, newbucket, newkey)
	retrievedjson := gocite.Passage{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	retrievedjson.Text.Brucheion = text //assuming that gocite.Passage.Text.Brucheion is meant to be the representation without newlines
	retrievedjson.Text.TXT = linetext   //gocite.Passage.Text.TXT is meant to be the representation with newlines
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
	log.Println("vars=[\"key\"]=" + vars["key"])
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	log.Println(req.Method)
	/*if r.Method != "POST" {
		io.WriteString(w, "Only POST is supported!")
		return
	}*/
	//log.Println(req.ParseForm())
	//log.Println(req.FormValue("text"))
	text := req.FormValue("text")
	log.Println("Debug: text=" + text)
	text = req.FormValue("text2")
	log.Println("Debug: text2=" + text)
	text = "BLABLABLA"
	log.Println("Debug: text=" + text)
	//test := req.FormValue("test")
	//log.Println("Debug: test=" + test)
	linetext := text
	//linetext := strings.Split(text, "\r\n")
	text = strings.Replace(text, "\r\n", "", -1)
	dbname := user + ".db"
	retrieveddata := BoltRetrieve(dbname, newbucket, newkey)
	retrievedjson := gocite.Passage{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	retrievedjson.Text.Brucheion = text //assuming that gocite.Passage.Text.Brucheion is meant to be the representation without newlines
	retrievedjson.Text.TXT = linetext   //gocite.Passage.Text.TXT is meant to be the representation with newlines
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
