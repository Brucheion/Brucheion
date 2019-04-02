package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/ThomasK81/gocite"

	"github.com/boltdb/bolt"

	"github.com/gorilla/mux"
)

//imageCollection is the container for image collections along with their URN and name as strings
type imageCollection struct {
	URN        string  `json:"urn"`
	Name       string  `json:"name"`
	Collection []image `json:"location"`
}

//newCITECollection extracts images from the mux variables in the *http.Request, joins them together
//in an imageCollection and passes it to newCollectionToDB to have it saved in the user database
func newCollection(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("newCollection", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	name := vars["name"]
	imageIDs := strings.Split(vars["urns"], ",")
	var collection imageCollection
	switch len(imageIDs) {
	case 0:
		io.WriteString(res, "failed")
		return
	case 1:
		urn := gocite.SplitCITE(imageIDs[0])
		switch {
		case urn.InValid:
			io.WriteString(res, "failed")
			return
		case urn.Object == "*":
			links, err := extractLinks(urn)
			if err != nil {
				io.WriteString(res, "failed")
			}
			for i := range links {
				collection.Collection = append(collection.Collection, image{External: false, Location: links[i]})
			}
		default:
			collection.Collection = append(collection.Collection, image{External: false, Location: imageIDs[0]})
		}
	default:
		for i := range imageIDs {
			urn := gocite.SplitCITE(imageIDs[i])
			switch {
			case urn.InValid:
				continue
			default:
				collection.Collection = append(collection.Collection, image{External: false, Location: imageIDs[i]})
			}
		}
	}
	newCollectionToDB(user, name, collection)
	io.WriteString(res, "success")
}

//newCollectiontoDB saves a new collection in a user database. Called by endpoint newCollection.
func newCollectionToDB(dbName, collectionName string, collection imageCollection) error {
	pwd, _ := os.Getwd()
	dbname := pwd + "/" + dbName + ".db"
	dbkey := []byte(collectionName)
	dbvalue, err := gobEncode(&collection)
	if err != nil {
		fmt.Println(err)
		return err
	}
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		return err
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("imgCollection"))
		if err != nil {
			fmt.Println(err)
			return err
		}
		val := bucket.Get(dbkey)
		if val != nil {
			fmt.Println("collection exists already")
			return errors.New("collection exists already")
		}
		err = bucket.Put(dbkey, dbvalue)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

//deleteCollection deletes the collection specified in the URL from the user database
func deleteCollection(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("deleteCollection", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	newkey := req.URL.Query().Get("name")
	newkey = strings.Replace(newkey, "\"", "", -1)
	dbname := user + ".db"
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	db.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("imgCollection")).Delete([]byte(newkey))
		if err != nil {
			fmt.Println(err)
			return err
		}
		return nil
	})
}