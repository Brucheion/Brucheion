package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/ThomasK81/gocite"
	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
)

//BoltData is the container for CITE data imported from CEX files and is used in LoadCEX
type BoltData struct {
	Bucket  []string // workurn
	Data    []BoltWork
	Catalog []BoltCatalog
}

//BoltWork is the container for BultURNs and their associated keys and is used in LoadCEX
type BoltWork struct {
	Key  []string // cts-node urn
	Data []BoltURN
}

//BoltCatalog contains all metadata of a CITE URN and is used in LoadCEX and page functions
type BoltCatalog struct {
	URN           string `json:"urn"`
	Citation      string `json:"citationScheme"`
	GroupName     string `json:"groupName"`
	WorkTitle     string `json:"workTitle"`
	VersionLabel  string `json:"versionLabel"`
	ExemplarLabel string `json:"exemplarLabel"`
	Online        string `json:"online"`
	Language      string `json:"language"`
}

//BoltURN is the container for a textpassage along with its URN, its image reference,
//and some information on preceding and anteceding works.
//Used for loading and saving CEX files, for pages, and for nodes
type BoltURN struct {
	URN      string   `json:"urn"`
	Text     string   `json:"text"`
	LineText []string `json:"linetext"`
	Previous string   `json:"previous"`
	Next     string   `json:"next"`
	First    string   `json:"first"`
	Last     string   `json:"last"`
	Index    int      `json:"sequence"`
	ImageRef []string `json:"imageref"`
}

//BoltJSON is a string representation of a JSON used in BoltRetrieve
type BoltJSON struct {
	JSON string
}

//cexMeta is the container for CEX metadata. Used for saving new URNs with newWork
//or changing metatdata with updateWorkMeta
type cexMeta struct {
	URN, CitationScheme, GroupName, WorkTitle, VersionLabel, ExemplarLabel, Online, Language string
}

//imageCollection is the container for image collections along with their URN and name as strings
type imageCollection struct {
	URN        string  `json:"urn"`
	Name       string  `json:"name"`
	Collection []image `json:"location"`
}

//image is the container for image metadata
type image struct {
	URN      string `json:"urn"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	License  string `json:"license"`
	External bool   `json:"external"`
	Location string `json:"location"`
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

//newCollectiontoDB saves a new collection in a user db. Called by endpoint newCollection
func newCollectiontoDB(dbName, collectionName string, collection imageCollection) error {
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

//newCITECollectionDB saves a new CITE collection with a specified name in the user database.
//Called by newCITECollection
func newCITECollectionDB(dbName, collectionName string) error {
	pwd, _ := os.Getwd()
	dbname := pwd + "/" + dbName + ".db"
	dbkey := []byte(collectionName)
	collection := imageCollection{}
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

func addtoCITECollection(dbName, collectionName string, newImage image) error {
	collection := imageCollection{}
	pwd, _ := os.Getwd()
	dbname := pwd + "/" + dbName + ".db"
	dbkey := []byte(collectionName)
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
		// fmt.Println("got", string(dbkey))

		if val != nil {
			collection, _ = gobDecodeImgCol(val)
		}
		found := false
		for coli, colv := range collection.Collection {
			if colv.URN == newImage.URN {
				found = true
				collection.Collection[coli] = newImage
			}
		}
		if !found {
			collection.Collection = append(collection.Collection, newImage)
			found = false
		}
		dbvalue, err2 := gobEncode(&collection)
		if err2 != nil {
			fmt.Println(err)
			return err
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

func newWorktoDB(dbName string, meta cexMeta) error {
	pwd, _ := os.Getwd()
	dbname := pwd + "/" + dbName + ".db"
	dbkey := []byte(meta.URN)
	dbvalue, err := gobEncode(&meta)
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		return err
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("meta"))
		if err != nil {
			return err
		}
		val := bucket.Get(dbkey)
		if val != nil {
			return errors.New("work exists already")
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

func updateWorkMeta(dbName string, meta cexMeta) error {
	pwd, _ := os.Getwd()
	dbname := pwd + "/" + dbName + ".db"
	dbkey := []byte(meta.URN)
	dbvalue, err := gobEncode(&meta)
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		return err
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("meta"))
		if err != nil {
			return err
		}
		val := bucket.Get(dbkey)
		if val == nil {
			return errors.New("work does not exist")
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

//gobEncode encodes an interface to a byte slice
func gobEncode(p interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(p)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

//gobDecodeImgCol decodes a byte slice to an imageCollection
func gobDecodeImgCol(data []byte) (imageCollection, error) {
	var p *imageCollection
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&p)
	if err != nil {
		return imageCollection{}, err
	}
	return *p, nil
}

//gobDecodePassage decodes a byte slice to a gocite.Passage
func gobDecodePassage(data []byte) (gocite.Passage, error) {
	var p *gocite.Passage
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&p)
	if err != nil {
		return gocite.Passage{}, err
	}
	return *p, nil
}

//BoltRetrieveFirstKey returns the first key in a specified bucket of
//a specified database as a string.
func BoltRetrieveFirstKey(dbname, bucket string) string {
	var result string
	if _, err := os.Stat(dbname); os.IsNotExist(err) {
		log.Println(err)
		return result
	}
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		return result
	}
	defer db.Close()
	// retrieve the data
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found", bucket)
		}
		c := bucket.Cursor()
		key, _ := c.First()
		result = string(key)
		return nil
	})
	return result
}

//BoltRetrieve retrieves the string data for the specified key in a specified bucket of
//a specified database as a string
func BoltRetrieve(dbname, bucket, key string) BoltJSON {
	var result BoltJSON
	if _, err := os.Stat(dbname); os.IsNotExist(err) {
		log.Println(err)
		return result
	}
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		return result
	}
	defer db.Close()
	// retrieve the data
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found", bucket)
		}
		val := bucket.Get([]byte(key))
		result.JSON = string(val)
		return nil
	})
	return result
}

//Buckets returns a slice of strings with the names of all buckets in a BoltDB.
func Buckets(dbname string) []string {
	var result []string
	if _, err := os.Stat(dbname); os.IsNotExist(err) {
		log.Println(err)
		return result
	}
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		return result
	}
	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
			result = append(result, string(name))
			return nil
		})
	})
	if err != nil {
		log.Println(err)
		return result
	}
	return result
}

func deleteBucket(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("deleteBucket", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	newbucket := vars["urn"]
	dbname := user + ".db"
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(newbucket))
		if err != nil {
			fmt.Println(err)
			return err
		}
		return nil
	})
}

func deleteNode(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("deleteNode", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	newkey := vars["urn"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	dbname := user + ".db"
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	db.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte(newbucket)).Delete([]byte(newkey))
		if err != nil {
			fmt.Println(err)
			return err
		}
		return nil
	})
	// Still to do: correct index, previous, next...
}
