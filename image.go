package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/boltdb/bolt"
)

func requestImgCollection(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := GetSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("requestImgCollection", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(res, req)
		return
	}

	response := JSONlist{}
	dbname := user + ".db"
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
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

func getImageInfo(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	//First get the session..
	session, err := GetSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("getImageInfo", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(res, req)
		return
	}

	retImage := imageCollection{}
	newImage := image{}
	collectionName := vars["name"]
	imageurn := vars["imageurn"]
	dbkey := []byte(collectionName)
	dbname := user + ".db"
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
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
	fmt.Println("request:", collectionName, collectionName)
	fmt.Println("answer:", newImage)
	resultJSON, _ := json.Marshal(newImage)
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(res, string(resultJSON))
}

func requestImgID(res http.ResponseWriter, req *http.Request) {
	//First get the session..
	session, err := GetSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("requestImgID", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(res, req)
		return
	}

	response := JSONlist{}
	collection := imageCollection{}
	vars := mux.Vars(req)
	name := vars["name"]
	dbname := user + ".db"
	dbkey := []byte(name)
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
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
