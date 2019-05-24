package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ThomasK81/gocite"

	"github.com/gorilla/mux"

	"github.com/boltdb/bolt"
)

//image is the container for image metadata
type image struct {
	URN      string `json:"urn"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	License  string `json:"license"`
	External bool   `json:"external"`
	Location string `json:"location"`
}

//JSONlist is a container for JSON items used for requests
type JSONlist struct {
	Item []string `json:"item"`
}

// requestImgCollection prints a list of the images contained in the image collection bucket
//in the user database.
func requestImgCollection(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("requestImgCollection", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	response := JSONlist{}
	dbname := user + ".db"
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		log.Println(fmt.Printf("requestImgCollection: error opening userDB: %s", err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("imgCollection"))
		if b == nil {
			return errors.New("failed to get bucket")
		}
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			response.Item = append(response.Item, string(k))
		}
		return nil
	})
	if err != nil {
		resultJSON, _ := json.Marshal(response)
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintln(res, string(resultJSON))
	}
	resultJSON, _ := json.Marshal(response)
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(res, string(resultJSON))
}

//getImageInfo prints the metadata of a specific image in the user database.
func getImageInfo(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("getImageInfo", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	retImage := imageCollection{}
	newImage := image{}
	collectionName := vars["name"]
	imageurn := vars["imageurn"]
	dbkey := []byte(collectionName)
	dbname := user + ".db"
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		log.Println(fmt.Printf("getImageInfo: error opening userDB: %s", err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("imgCollection"))
		if b == nil {
			return errors.New("failed to get bucket")
		}
		val := b.Get(dbkey)
		// fmt.Println("got", string(dbkey))
		retImage, _ = gobDecodeImgCol(val)
		for _, v := range retImage.Collection {
			if v.URN == imageurn {
				newImage = v
			}
		}
		return nil
	})
	if err != nil {
		resultJSON, _ := json.Marshal(newImage)
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintln(res, string(resultJSON))
	}
	fmt.Println("request:", collectionName, collectionName) //why twice?
	fmt.Println("answer:", newImage)
	resultJSON, _ := json.Marshal(newImage)
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(res, string(resultJSON))
}

func requestImgID(res http.ResponseWriter, req *http.Request) {
	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("requestImgID", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	response := JSONlist{}
	collection := imageCollection{}
	vars := mux.Vars(req)
	name := vars["name"]
	dbname := user + ".db"
	dbkey := []byte(name)
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		log.Println(fmt.Printf("requestImgID: error opening userDB: %s", err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("imgCollection"))
		if bucket == nil {
			return errors.New("failed to get bucket")
		}
		val := bucket.Get(dbkey)
		if val == nil {
			return errors.New("failed to retrieve value")
		}
		collection, err = gobDecodeImgCol(val)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		resultJSON, _ := json.Marshal(response)
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintln(res, string(resultJSON))
	}
	for i := range collection.Collection {
		response.Item = append(response.Item, collection.Collection[i].URN)
	}
	resultJSON, _ := json.Marshal(response)
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(res, string(resultJSON))
}

// SaveImageRef parses and updated image references from the http.Request
//and saves it to the corresponding URN bucket in the user database
func SaveImageRef(res http.ResponseWriter, req *http.Request) {

	//DEBUGGING
	// fmt.Println(r.Method)
	// if r.Method != "POST" {
	// 	vars := mux.Vars(r)
	// 	newkey := vars["key"]
	// 	imagerefstr := r.FormValue("text")
	// 	fmt.Println(newkey, imagerefstr)
	// 	io.WriteString(w, "Only POST is supported!")
	// 	return
	// }
	// fmt.Println(r.ParseForm())
	// fmt.Println(r.FormValue("text"))

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("SaveImageRef", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	newkey := vars["key"]
	imagerefstr := vars["updated"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	// imagerefstr := r.FormValue("text")
	imageref := strings.Split(imagerefstr, "+")
	dbname := user + ".db"
	retrieveddata, _ := BoltRetrieve(dbname, newbucket, newkey)
	retrievedjson := gocite.Passage{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	var textareas []gocite.Triple
	for i := range imageref {
		textareas = append(textareas, gocite.Triple{Subject: newkey,
			Verb:   "urn:cite2:dse:verbs.v1:appears_on",
			Object: imageref[i]})
	}
	retrievedjson.ImageLinks = textareas
	newnode, _ := json.Marshal(retrievedjson)
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		log.Println(fmt.Printf("SaveImageRef: error opening userDB: %s", err))
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
			fmt.Println(err)
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
	reDir := "/view/" + newkey
	http.Redirect(res, req, reDir, http.StatusFound)
}
