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
	"time"

	"github.com/ThomasK81/gocite"

	"github.com/boltdb/bolt"

	"github.com/gorilla/mux"
)

//BoltData is the container for CITE data imported from CEX files and is used in LoadCEX
type BoltData struct {
	Bucket  []string // workurn
	Data    []gocite.Work
	Catalog []BoltCatalog
}

//BoltWork is the container for BultURNs and their associated keys and is used in LoadCEX
type BoltWork struct {
	Key  []string // cts-node urn
	Data []BoltURN
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

//gobEncode encodes an interface to a byte slice, to be saved in the database
func gobEncode(p interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(p)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

//gobDecodeImgCol decodes a byte slice from the database to an imageCollection
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

//gobDecodePassage decodes a byte slice from the database to a gocite.Passage
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

//openBoltDB returns an opened Bolt Database for given dbName.
func openBoltDB(dbName string) (*bolt.DB, error) {
	db, err := bolt.Open(dbName, 0600, &bolt.Options{Timeout: 30 * time.Second}) //open DB with - wr- --- ---
	if err != nil {
		return nil, err
	}
	//fmt.Println("DB opened")
	return db, nil
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

// newCITECollectionToDB saves a new CITE collection with a specified name in the user database.
//Called by newCITECollection
func newCITECollectionToDB(dbName, collectionName string) error {
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

// addImageToCITECollection adds image metadata to the specified collection
//in the bucket imgCollection in a user database. Called by addCITE
func addImageToCITECollection(dbName, collectionName string, newImage image) error {
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
		val := bucket.Get(dbkey) //search for collection in bucket
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

//newWorkToDB saves cexMeta data to the meta bucket in the user database
//called by newWork
func newWorkToDB(dbName string, meta cexMeta) error {
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
			return errors.New("work already exists")
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

//updateWorkMeta saves cexMeta data for an already existing key in the meta bucket
//in the user database. Seems not to be called yet.
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
			return errors.New("work does not exist yet")
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

// BoltRetrieve retrieves the string data (as BoltJSON) for the specified key
//in the specified bucket of the specified database as a BoltJSON
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

//deleteBucket deletes a bucket with the name of a specified URN
func deleteBucket(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("deleteBucket", session)
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

//deleteNode deletes the specified bucket (that is related to a certain node?)
func deleteNode(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("deleteNode", session)
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
