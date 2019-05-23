package main

import (
	"fmt"
	"log"
	"net/http"

	"gociteDev/gocite"
	/*
		"encoding/json"
		"strconv"
		"strings"

		"github.com/boltdb/bolt"

		"github.com/gorilla/mux"

		"gociteDev/gocite"*/)

//AddFirstNode adds a new passage at the beginning of the work
func AddFirstNode(res http.ResponseWriter, req *http.Request) {
	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("AddFirstNode", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	dbName := user + ".db"
	passage, err := BoltRetrievePassage(dbName, "urn:cts:sktlit:skt0001.nyaya002.msC3D:", "3.1.44")

	log.Printf("Passage received: %s \nIndex: %d\nPrev: %d\nNext: %d\n",
		passage.PassageID, passage.Index, passage.Prev.Index, passage.Next.Index)

	work, err := BoltRetrieveWork(dbName, "urn:cts:sktlit:skt0001.nyaya002.msC3D:")
	if err == nil {
		log.Printf(" Work received: WorkID: %s\nFirst.Index: %d\nFirst.PassageID: %s\nLast.Index: %d\nLast.PassageID: %s\n\n",
			work.WorkID,
			work.First.Index, work.First.PassageID,
			work.Last.Index, work.Last.PassageID)
		for i := range work.Passages {
			log.Println(fmt.Printf(" Work received: PassageID: %s\n Index: %d\nPrev.Index: %d\nPrev.PassageID: %s\nNext.Index: %d\nNext.PassageID: %s\n\n",
				work.Passages[i].PassageID, work.Passages[i].Index,
				work.Passages[i].Prev.Index, work.Passages[i].Prev.PassageID,
				work.Passages[i].Next.Index, work.Passages[i].Next.PassageID))
		}
	} else {
		log.Println(err)
	}

	work, err = gocite.SortPassages(work)
	if err == nil {
		log.Printf(" SORTED AGAIN: WorkID: %s\nFirst.Index: %d\nFirst.PassageID: %s\nLast.Index: %d\nLast.PassageID: %s\n\n",
			work.WorkID,
			work.First.Index, work.First.PassageID,
			work.Last.Index, work.Last.PassageID)
		for i := range work.Passages {
			log.Println(fmt.Printf(" Work received: PassageID: %s\n Index: %d\nPrev.Index: %d\nPrev.PassageID: %s\nNext.Index: %d\nNext.PassageID: %s\n\n",
				work.Passages[i].PassageID, work.Passages[i].Index,
				work.Passages[i].Prev.Index, work.Passages[i].Prev.PassageID,
				work.Passages[i].Next.Index, work.Passages[i].Next.PassageID))
		}
	} else {
		log.Println(err)
	}
	/*result2, err := gocite.SortPassages(work)
	log.Printf(" SORTED result: WorkID: %s\nFirst.Index: %d\nFirst.PassageID: %s\nLast.Index: %d\nLast.PassageID: %s\n\n",
		result2.WorkID,
		result2.First.Index, result2.First.PassageID,
		result2.Last.Index, result2.Last.PassageID)
	for i := range result2.Passages {
		log.Println(fmt.Printf(" SORTED result: PassageID: %s\n Index: %d\nPrev.Index: %d\nPrev.PassageID: %s\nNext.Index: %d\nNext.PassageID: %s\n\n",
			result2.Passages[i].PassageID, result2.Passages[i].Index,
			result2.Passages[i].Prev.Index, result2.Passages[i].Prev.PassageID,
			result2.Passages[i].Next.Index, result2.Passages[i].Next.PassageID))*/
}

/*
		var first, last, prev, next string

		vars := mux.Vars(req)

		/*
		    	var texturns, texts, previouss, nexts, firsts, lasts []string
		    	var imagerefs, linetexts [][]string
		    	var indexs []int
		    	vars := mux.Vars(req)
		    	newkey := vars["key"]
		    	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
		    	dbname := user + ".db"
		    	retrieveddata := BoltRetrieve(dbname, newbucket, newkey)
		    	retrievednodejson := BoltURN{}
		    	json.Unmarshal([]byte(retrieveddata.JSON), &retrievednodejson)
		   	marker := retrievednodejson.First
		    	retrieveddata = BoltRetrieve(dbname, newbucket, marker)
		   	retrievednodejson = BoltURN{}
		    	json.Unmarshal([]byte(retrieveddata.JSON), &retrievednodejson)
		    	bookmark := retrievednodejson.Index
		    	lastnode := false
		    	if retrievednodejson.Last == retrievednodejson.URN {
		    		lastnode = true
		    	}
		    	db, err := openBoltDB(dbname) //open bolt DB using helper function
		    	if err != nil {
		    		fmt.Printf("Error opening userDB: %s", err)
		    		http.Error(res, err.Error(), http.StatusInternalServerError)
		    		return
		    	}
		    	defer db.Close()


}*/

/*
 func AddFirstNodeOLD(res http.ResponseWriter, req *http.Request) {

 	//First get the session..
 	session, err := getSession(req)
 	if err != nil {
 		http.Error(res, err.Error(), http.StatusInternalServerError)
 		return
 	}

 	//..and check if user is logged in.
 	user, message, loggedin := testLoginStatus("AddFirstNode", session)
 	if loggedin {
 		log.Println(message)
 	} else {
 		log.Println(message)
 		Logout(res, req)
 		return
 	}


 	var texturns, texts, previouss, nexts, firsts, lasts []string
 	var imagerefs, linetexts [][]string
 	var indexs []int
 	vars := mux.Vars(req)
 	newkey := vars["key"]
 	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
 	dbname := user + ".db"
 	retrieveddata := BoltRetrieve(dbname, newbucket, newkey)
 	retrievednodejson := BoltURN{}
 	json.Unmarshal([]byte(retrieveddata.JSON), &retrievednodejson)
	marker := retrievednodejson.First
 	retrieveddata = BoltRetrieve(dbname, newbucket, marker)
	retrievednodejson = BoltURN{}
 	json.Unmarshal([]byte(retrieveddata.JSON), &retrievednodejson)
 	bookmark := retrievednodejson.Index
 	lastnode := false
 	if retrievednodejson.Last == retrievednodejson.URN {
 		lastnode = true
 	}
 	db, err := openBoltDB(dbname) //open bolt DB using helper function
 	if err != nil {
 		fmt.Printf("Error opening userDB: %s", err)
 		http.Error(res, err.Error(), http.StatusInternalServerError)
 		return
 	}
 	defer db.Close()

 	db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
 		b := tx.Bucket([]byte(newbucket))
		c := b.Cursor()

 		for k, v := c.First(); k != nil; k, v = c.Next() {
 			retrievedjson := BoltURN{}
 			json.Unmarshal([]byte(v), &retrievedjson)
 			ctsurn := retrievedjson.URN
 			text := retrievedjson.Text
 			linetext := retrievedjson.LineText
 			previous := retrievedjson.Previous
 			next := retrievedjson.Next
 			imageref := retrievedjson.ImageRef
 			last := retrievedjson.Last
 			index := retrievedjson.Index
 			newfirst := newbucket + "newNode" + strconv.Itoa(bookmark)

 			switch {
 			case index < bookmark:
 				texturns = append(texturns, ctsurn)
 				texts = append(texts, text)
 				linetexts = append(linetexts, linetext)
 				previouss = append(previouss, previous)
 				nexts = append(nexts, next)
 				firsts = append(firsts, newfirst)
 				switch lastnode {
 				case false:
 					lasts = append(lasts, last)
 				case true:
 					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
 					lasts = append(lasts, newlast)
 				}
				indexs = append(indexs, index)
				imagerefs = append(imagerefs, imageref)
 			case index > bookmark+1:
 				newindex := index + 1
 				texturns = append(texturns, ctsurn)
 				texts = append(texts, text)
 				linetexts = append(linetexts, linetext)
 				previouss = append(previouss, previous)
				nexts = append(nexts, next)
 				firsts = append(firsts, newfirst)
 				switch lastnode {
 				case false:
					lasts = append(lasts, last)
 				case true:
 					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
 					lasts = append(lasts, newlast)
 				}
 				indexs = append(indexs, newindex)
 				imagerefs = append(imagerefs, imageref)
 			case index == bookmark:
 				newnode := newbucket + "newNode" + strconv.Itoa(index)
 				newindex := index + 1

 				texturns = append(texturns, newnode)
 				texts = append(texts, "")
 				linetexts = append(linetexts, []string{})
				previouss = append(previouss, newfirst)
 				nexts = append(nexts, ctsurn)
 				firsts = append(firsts, newfirst)
				switch lastnode {
 				case false:
 					lasts = append(lasts, last)
 				case true:
 					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
 					lasts = append(lasts, newlast)
 				}
 				indexs = append(indexs, index)
 				imagerefs = append(imagerefs, []string{})

 				texturns = append(texturns, ctsurn)
 				texts = append(texts, text)
 				linetexts = append(linetexts, linetext)
 				previouss = append(previouss, newfirst)
 				nexts = append(nexts, next)
 				firsts = append(firsts, newfirst)
				switch lastnode {
 				case false:
					lasts = append(lasts, last)
 				case true:
 					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
 					lasts = append(lasts, newlast)
 				}
				indexs = append(indexs, newindex)
 				imagerefs = append(imagerefs, imageref)
 			case index == bookmark+1:
 				newnode := newbucket + "newNode" + strconv.Itoa(bookmark)
				newindex := index + 1
 				texturns = append(texturns, ctsurn)
 				texts = append(texts, text)
 				linetexts = append(linetexts, linetext)
				previouss = append(previouss, newnode)
 				nexts = append(nexts, next)
 				firsts = append(firsts, newfirst)
 				switch lastnode {
 				case false:
 					lasts = append(lasts, last)
 				case true:
 					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
 					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, newindex)
 				imagerefs = append(imagerefs, imageref)
 			}
 		}
 		return nil
 	})

 	var bolturns []gocite.Passage
 	lastTextID := len(texturns)-1
 	for i := range texturns {
 		prevEx := true
 		nextEx := true
 		if i == 0 {prevEx = false}
 		if i == lastTextID {nextEx = false}
 		bolturns = append(bolturns, gocite.Passage{PassageID: texturns[i],
 			Range: false,
 			Text:     gocite.EncText{Brucheion: texts[i],
 			TXT: linetexts[i]},
 			Previous: gocite.PassLoc{Exists: prevEx,
 				PassageID: previouss[i],
 				index = i - 1},
 			Next:     gocite.PassLoc{Exists: nextEx,
 				PassageID: nexts[i],
 				index = i + 1},
 			First:    gocite.PassLoc{Exists: true,
 				PassageID: firsts[i],
 				index = 0},
 			Last:     gocite.PassLoc{Exists: true,
 				PassageID: lasts[i],
 				index = lastTextID},
 			Index:    indexs[i],
 			ImageRef: imagerefs[i]})
 	}
 	for i := range bolturns {
 		newkey := texturns[i]
 		newnode, _ := json.Marshal(bolturns[i])
 		key := []byte(newkey)
 		value := []byte(newnode)
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
 	}
 }
*/

/* Has to be rebuild anyway. Overhauled until mark

// AddNodeAfter adds
func AddNodeAfter(res http.ResponseWriter, req *http.Request) {
	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("AddNodeAfter", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	var texturns, texts, previouss, nexts, firsts, lasts []string
	//var imagerefs, linetexts [][]string
	var imagerefs [][]string
	var indexs []int
	vars := mux.Vars(req)
	PassageURNString := vars["key"]
	URNString := strings.Join(strings.Split(PassageURNString, ":")[0:4], ":") + ":" //
	passageIdentifier := strings.Split(PassageURNString, ":")[5]
	dbname := user + ".db"
	retrievedPassage := BoltRetrievePassage(dbname, URNString, passageIdentifier)
	//retrievedData, _ := BoltRetrieve(dbname, URNString, PassageURNString)
	//retrievednodejson := BoltURN{}
	//retrievedPassage := gocite.Passage{}
	//json.Unmarshal([]byte(retrievedData.JSON), &retrievedPassage)
	bookmark := retrievedPassage.Index
	lastnode := false
	//if retrievednodejson.Last == retrievednodejson.URN {
	if BoltRetrieveWork(dbname, URNString).Last.PassageID == retrievedPassage.PassageID {
		lastnode = true
	}
	*** OVERHAULED UNTIL HERE ***


	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		log.Println(fmt.Printf("AddNodeAfter: error opening userDB: %s", err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		bucket := tx.Bucket([]byte(URNString))

		cursor := bucket.Cursor()

		for key, value := cursor.First(); key != nil; key, value = cursor.Next() {
			retrievedjson := BoltURN{}
			json.Unmarshal([]byte(v), &retrievedjson)
			ctsurn := retrievedjson.URN
			text := retrievedjson.Text
			linetext := retrievedjson.LineText
			previous := retrievedjson.Previous
			next := retrievedjson.Next
			first := retrievedjson.First
			imageref := retrievedjson.ImageRef
			last := retrievedjson.Last
			index := retrievedjson.Index

			retrievedjson := gocite.Passage{}
			json.Unmarshal([]byte(value), &retrievedjson)
			ctsurn := retrievedjson.PassageID
			text := retrievedjson.Text.TXT
			//linetext is being retired and replaced with to gocite.Text
			previous := retrievedjson.Prev.PassageID
			next := retrievedjson.Next.PassageID
			first := retrievedjson.First.PassageID
			last := retrievedjson.Last.PassageID
			imageref := []string{}
			for _, tmp := range retrievedjson.ImageLinks {
				imageref = append(imageref, tmp.Object)
			}
			index := retrievedjson.Index

			switch {
			case index < bookmark:
				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				//linetexts = append(linetexts, linetext)
				previouss = append(previouss, previous)
				nexts = append(nexts, next)
				firsts = append(firsts, first)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := URNString + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, index)
				imagerefs = append(imagerefs, imageref)
			case index > bookmark+1:
				newindex := index + 1
				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				//linetexts = append(linetexts, linetext)
				previouss = append(previouss, previous)
				nexts = append(nexts, next)
				firsts = append(firsts, first)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := URNString + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, newindex)
				imagerefs = append(imagerefs, imageref)
			case index == bookmark:
				newnode := URNString + "newNode" + strconv.Itoa(index)
				newindex := index + 1

				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				//linetexts = append(linetexts, linetext)
				previouss = append(previouss, previous)
				nexts = append(nexts, newnode)
				firsts = append(firsts, first)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := URNString + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, index)
				imagerefs = append(imagerefs, imageref)

				texturns = append(texturns, newnode)
				texts = append(texts, "")
				//linetexts = append(linetexts, []string{})
				previouss = append(previouss, ctsurn)
				nexts = append(nexts, next)
				firsts = append(firsts, first)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := URNString + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, newindex)
				imagerefs = append(imagerefs, []string{})
			case index == bookmark+1:
				newnode := URNString + "newNode" + strconv.Itoa(bookmark)
				newindex := index + 1
				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				//linetexts = append(linetexts, linetext)
				previouss = append(previouss, newnode)
				nexts = append(nexts, next)
				firsts = append(firsts, first)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := URNString + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, newindex)
				imagerefs = append(imagerefs, imageref)
			}
		}
		return nil
	})

	//var bolturns []BoltURN
	var gocitePassages []gocite.Passage
	for i := range texturns {
		bolturns = append(bolturns, BoltURN{URN: texturns[i],
		Text:     texts[i],
		//LineText: linetexts[i],
		Previous: previouss[i],
		Next:     nexts[i],
		First:    firsts[i],
		Last:     lasts[i],
		Index:    indexs[i],
		ImageRef: imagerefs[i]})

		text := gocite.EncText{}
		text.TXT = texts[i]
		var previous, next, first, last gocite.PassLoc
		previous.PassageID = previouss[i]
		next.PassageID = nexts[i]
		first.PassageID = firsts[i]
		last.PassageID = lasts[i]

		gocitePassages = append(gocitePassages, gocite.Passage{
			PassageID: texturns[i],
			Text:      text,
			Prev:      previous,
			Next:      next,
			First:     first,
			Last:      last,
			Index:     indexs[i]})
		//ImageRef:  imagerefs[i]}) //What to do with the imagerefs?
	}
	//for i := range bolturns {
	for i := range gocitePassages {
		newkey := texturns[i]
		newnode, _ := json.Marshal(gocitePassages[i])
		key := []byte(newkey)
		value := []byte(newnode)
		err = db.Update(func(tx *bolt.Tx) error {
			bucket, err := tx.CreateBucketIfNotExists([]byte(URNString))
			if err != nil {
				return err
			}

			err = bucket.Put(key, value)
			if err != nil {
				return err
			}
			return nil
		})
	}
}
*/
