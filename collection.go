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
	user, message, loggedin := testLoginStatus("newCollection", session)
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
			log.Println(fmt.Errorf("newCollection: Error saving Image collection %s in %s.db: URN invalid", name, user))
			io.WriteString(res, "Import of image collection "+name+" failed: URN invalid.")
			return
		case urn.Object == "*":
			links, err := extractLinks(urn)
			if err != nil {
				log.Println(fmt.Errorf("newCollection: Error saving Image collection %s in %s.db: %s", name, user, err))
				io.WriteString(res, "Import of image collection "+name+" failed: extracting links failed.")
			}
			for i := range links {
				collection.Collection = append(collection.Collection, image{URN: links[i],
					Name:     links[i],
					Protocol: "localDZ",
					Location: links[i]})
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
	err = newCollectionToDB(user, name, collection)

	if err != nil {
		log.Println(fmt.Errorf("newCollectionToDB: Error saving Image collection %s in %s.db: %s", name, user, err))
		io.WriteString(res, "Import of image collection "+name+" failed")
	}
	log.Println("newCollection: Image collection " + name + "saved in " + user + ".db successfully.")
	io.WriteString(res, "Image collection "+name+" imported successfully.")
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
		log.Println(fmt.Printf("newCollectionToDB: error opening userDB: %s", err))
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
			return errors.New("collection already exists")
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
// localhost:7000/deleteCollection/?name=urn:cite2:nyaya:Awimg.positive:
func deleteCollection(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("deleteCollection", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	newkey := req.URL.Query().Get("name")
	log.Println(newkey)
	newkey = strings.Replace(newkey, "\"", "", -1)
	log.Println(newkey)
	dbname := user + ".db"
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		log.Println(fmt.Printf("deleteCollection: error opening userDB: %s", err))
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
