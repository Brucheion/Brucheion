package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
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
		log.Println(fmt.Printf("Buckets: error opening userDB: %s", err))
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
		log.Println(fmt.Printf("Error opening userDB: %s", err))
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
		log.Println(fmt.Printf("addImageToCITECollection: error opening userDB: %s", err))
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
			//found = false //ineffectual assignment to found: found is false already
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
	//dbvalue, err := gobEncode(&meta) //ineffectual assignment to err: nothing is done with err before it is overwritten
	dbvalue, _ := gobEncode(&meta)
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		log.Println(fmt.Printf("newWorkToDB: error opening userDB: %s", err))
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
//in the user database. Seems not to be called yet. (not in use) (deprecated?)
func updateWorkMeta(dbName string, meta cexMeta) error {
	pwd, _ := os.Getwd()
	dbname := pwd + "/" + dbName + ".db"
	dbkey := []byte(meta.URN)
	//dbvalue, err := gobEncode(&meta) //ineffectual assignment to err: nothing is done with err before it is overwritten
	dbvalue, _ := gobEncode(&meta)
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		log.Println(fmt.Printf("updateWorkMeta: error opening userDB: %s", err))
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
//a specified database as a string. (deprecated)
func BoltRetrieveFirstKey(dbname, bucketName string) (string, error) {
	var result string
	if _, err := os.Stat(dbname); os.IsNotExist(err) {
		log.Println(err)
		return result, err
	}
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		log.Println(fmt.Printf("BoltRetrieveFirstKey: error opening userDB: %s", err))
		return result, err
	}
	defer db.Close()
	// retrieve the data
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found", bucketName)
		}
		c := bucket.Cursor()
		key, _ := c.First()
		result = string(key)
		return nil
	})
	return result, err
}

// BoltRetrievePassage retrieves a Passage from its bucket as a gocite.Passage object
func BoltRetrievePassage(dbName, workName, passageIdentifiers string) (gocite.Passage, error) {
	var result gocite.Passage
	if _, err := os.Stat(dbName); os.IsNotExist(err) {
		log.Println(err)
		return result, err
	}
	db, err := openBoltDB(dbName)
	if err != nil {
		log.Println(fmt.Printf("BoltRetrieve: error opening userDB: %s", err))
		return result, err
	}
	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(workName))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found", workName)
		}
		buffer := bucket.Get([]byte(workName + passageIdentifiers))
		err := json.Unmarshal(buffer, &result) //unmarshal the buffer and save the gocite.Passage
		if err != nil {
			log.Println(fmt.Printf("Error unmarshalling work: %s", err))
			return (err)
		}
		return nil
	})
	return result, err
}

//BoltRetrieveWork retrieves an entire work from the users database as an (ordered) gocite.Work object
func BoltRetrieveWork(dbName, workID string) (gocite.Work, error) {
	var result gocite.Work
	if _, err := os.Stat(dbName); os.IsNotExist(err) {
		log.Println(err)
		return result, err
	}
	db, err := openBoltDB(dbName)
	if err != nil {
		log.Printf("BoltRetrieve: error opening userDB: %s\n", err)
		return result, err
	}
	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(workID))
		if bucket == nil {
			return fmt.Errorf("BoltRetrieveWork: bucket %q not found", workID)
		}
		result.WorkID = workID
		cursor := bucket.Cursor()

		for key, value := cursor.First(); key != nil; key, value = cursor.Next() {
			var passage gocite.Passage
			err := json.Unmarshal(value, &passage) //unmarshal the buffer and save the gocite.Passage
			if err != nil {
				log.Println(fmt.Printf("BoltRetrieveWork: Error unmarshalling Passage: %s", err))
				return fmt.Errorf("BoltRetrieveWork: Error unmarshalling Passage: %s", err)
			}
			//log.Println("Adding " + passage.PassageID + " to work")
			if passage.PassageID != "" {
				result.Passages = append(result.Passages, passage)
			}
		}
		return nil
	})
	/*for i := range result.Passages {
		log.Println(fmt.Printf("result: received: %s \nIndex: %d\nFirst: %d\nLast: %d\nPrev: %d\nNext: %d\n",
			result.Passages[i].PassageID, result.Passages[i].Index, result.Passages[i].First.Index,
			result.Passages[i].Last.Index, result.Passages[i].Prev.Index, result.Passages[i].Next.Index))
	}*/
	result2 := gocite.SortPassages(result)
	for i := range result2.Passages {
		log.Println(fmt.Printf(" SORTED result: PassageID: %s\n Index: %d\nFirst.Index: %d\nFirst.PassageID: %s\nLast.Index: %d\nLast.PassageID: %s\nPrev.Index: %d\nPrev.PassageID: %s\nNext.Index: %d\nNext.PassageID: %s\n\n",
			result2.Passages[i].PassageID, result2.Passages[i].Index,
			result2.Passages[i].First.Index, result2.Passages[i].First.PassageID,
			result2.Passages[i].Last.Index, result2.Passages[i].Last.PassageID,
			result2.Passages[i].Prev.Index, result2.Passages[i].Prev.PassageID,
			result2.Passages[i].Next.Index, result2.Passages[i].Next.PassageID))
	}
	return result, err

}

// BoltRetrieve retrieves the string data (as BoltJSON) for the specified key
//in the specified bucket of the specified database as a BoltJSON (deprecated)
func BoltRetrieve(dbname, bucketName, key string) (BoltJSON, error) {
	var result BoltJSON
	if _, err := os.Stat(dbname); os.IsNotExist(err) {
		log.Println(err)
		return result, err
	}
	db, err := openBoltDB(dbname)
	if err != nil {
		log.Println(fmt.Printf("BoltRetrieve: error opening userDB: %s", err))
		return result, err
	}
	defer db.Close()
	// retrieve the data
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found", bucketName)
		}
		val := bucket.Get([]byte(key))
		result.JSON = string(val)
		return nil
	})
	return result, err
}

//deleteBucket deletes a bucket with the name of a specified URN
//(unused) (deprecated?)
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
		log.Println(fmt.Printf("deleteBucket: error opening userDB: %s", err))
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
//(unused) (deprecated)
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
