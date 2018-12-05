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

type BoltData struct {
	Bucket  []string // workurn
	Data    []BoltWork
	Catalog []BoltCatalog
}

type BoltWork struct {
	Key  []string // cts-node urn
	Data []BoltURN
}

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

type BoltJSON struct {
	JSON string
}

type cexMeta struct {
	URN, CitationScheme, GroupName, WorkTitle, VersionLabel, ExemplarLabel, Online, Language string
}

type imageCollection struct {
	URN        string  `json:"urn"`
	Name       string  `json:"name"`
	Collection []image `json:"location"`
}

type image struct {
	URN      string `json:"urn"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	License  string `json:"license"`
	External bool   `json:"external"`
	Location string `json:"location"`
}

func deleteCollection(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("deleteCollection", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	newkey := r.URL.Query().Get("name")
	newkey = strings.Replace(newkey, "\"", "", -1)
	dbname := user + ".db"
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

func newCollectiontoDB(dbName, collectionName string, collection imageCollection) error {
	pwd, _ := os.Getwd()
	dbname := pwd + "/" + dbName + ".db"
	dbkey := []byte(collectionName)
	dbvalue, err := gobEncode(&collection)
	if err != nil {
		fmt.Println(err)
		return err
	}
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
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
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
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
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
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
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
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
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
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

func gobEncode(p interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(p)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

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

func BoltRetrieveFirstKey(dbname, bucket string) string {
	var result string
	if _, err := os.Stat(dbname); os.IsNotExist(err) {
		log.Println(err)
		return result
	}
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		return result
	}
	defer db.Close()
	// retrieve the data
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucket))
		if bucket == nil {
			return fmt.Errorf("Bucket %q not found!", bucket)
		}
		c := bucket.Cursor()
		key, _ := c.First()
		result = string(key)
		return nil
	})
	return result
}

func BoltRetrieve(dbname, bucket, key string) BoltJSON {
	var result BoltJSON
	if _, err := os.Stat(dbname); os.IsNotExist(err) {
		log.Println(err)
		return result
	}
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
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

// Buckets returns a slice of strings with the names of all buckets in a BoltDB.
func Buckets(dbname string) []string {
	var result []string
	if _, err := os.Stat(dbname); os.IsNotExist(err) {
		log.Println(err)
		return result
	}
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
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

func deleteBucket(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("deleteCollection", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	vars := mux.Vars(r)
	newbucket := vars["urn"]
	dbname := user + ".db"
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

func deleteNode(w http.ResponseWriter, r *http.Request) {

	//First get the session..
	session, err := GetSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("deleteCollection", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(w, r)
	}

	vars := mux.Vars(r)
	newkey := vars["urn"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	dbname := user + ".db"
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
