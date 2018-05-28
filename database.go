package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

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

func BoltRetrieveFirstKey(path, bucket string) string {
	var result string
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Println(err)
		return result
	}

	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		log.Println(err)
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

func BoltRetrieve(path, bucket, key string) BoltJSON {
	var result BoltJSON
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Println(err)
		return result
	}

	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		log.Println(err)
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

// Buckets prints a list of all buckets.
func Buckets(path string) []string {
	var result []string
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Println(err)
		return result
	}

	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		log.Println(err)
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
	vars := mux.Vars(r)
	user := vars["user"]
	newbucket := vars["urn"]
	dbname := user + ".db"
	db, err := bolt.Open(dbname, 0644, nil)
	if err != nil {
		log.Fatal(err)
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
	vars := mux.Vars(r)
	user := vars["user"]
	newkey := vars["urn"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	dbname := user + ".db"
	db, err := bolt.Open(dbname, 0644, nil)
	if err != nil {
		log.Fatal(err)
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
