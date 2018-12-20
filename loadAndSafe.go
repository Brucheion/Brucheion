package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/boltdb/bolt"

	"github.com/gorilla/mux"

	"github.com/ThomasK81/gocite"
)

func newCITECollection(res http.ResponseWriter, req *http.Request) {
	//First get the session..
	session, err := GetSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("newCITECollection", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(res, req)
	}

	vars := mux.Vars(req)
	name := vars["name"] //the name of the new CITE collection
	newCITECollectionDB(user, name)
	io.WriteString(res, "success")
}

func newCollection(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := GetSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("newCollection", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(res, req)
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
	newCollectiontoDB(user, name, collection)
	io.WriteString(res, "success")
}

func newWork(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := GetSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("newWork", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(res, req)
	}

	if req.Method == "GET" {
		varmap := map[string]interface{}{
			"user": user,
			"port": config.Port,
		}
		t, _ := template.ParseFiles("tmpl/newWork.html")
		t.Execute(res, varmap)
	} else {
		req.ParseForm()
		// logic part of log in
		workurn := req.Form["workurn"][0]
		scheme := req.Form["scheme"][0]
		group := req.Form["workgroup"][0]
		title := req.Form["title"][0]
		version := req.Form["version"][0]
		exemplar := req.Form["exemplar"][0]
		online := req.Form["online"][0]
		language := req.Form["language"][0]
		newWork := cexMeta{URN: workurn, CitationScheme: scheme, GroupName: group, WorkTitle: title, VersionLabel: version, ExemplarLabel: exemplar, Online: online, Language: language}
		fmt.Println(newWork)
		err := newWorktoDB(user, newWork)
		if err != nil {
			io.WriteString(res, "failed")
		} else {
			io.WriteString(res, "Success")
		}
	}
}

func newText(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := GetSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("newText", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(res, req)
	}

	vars := mux.Vars(req)
	newkey := vars["key"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	dbname := user + ".db"
	retrievedjson := BoltURN{}
	retrievedjson.URN = newkey
	newnode, _ := json.Marshal(retrievedjson)
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
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
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
	http.Redirect(res, req, "/view/"+newkey, http.StatusFound)
}

func addCITE(res http.ResponseWriter, req *http.Request) {
	//First get the session..
	session, err := GetSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("addCITE", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(res, req)
	}

	// /thomas/addtoCITE?name="test"&urn="test"&internal="false"&protocol="static&location="https://digi.vatlib.it/iiifimage/MSS_Barb.lat.4/Barb.lat.4_0015.jp2/full/full/0/native.jpg"
	name := req.URL.Query().Get("name")
	name = strings.Replace(name, "\"", "", -1)
	imageurn := req.URL.Query().Get("urn")
	imageurn = strings.Replace(imageurn, "\"", "", -1)
	location := req.URL.Query().Get("location")
	location = strings.Replace(location, "\"", "", -1)
	// fmt.Println(location)
	protocol := req.URL.Query().Get("protocol")
	protocol = strings.Replace(protocol, "\"", "", -1)
	externalstr := req.URL.Query().Get("external")
	externalstr = strings.Replace(externalstr, "\"", "", -1)
	external := false
	if externalstr == "true" {
		external = true
	}
	newimage := image{URN: imageurn, External: external, Protocol: protocol, Location: location}
	// fmt.Println(user, name, newimage)
	addtoCITECollection(user, name, newimage)
	io.WriteString(res, "success")
}

//LoadCEX loads a CEX file, parses it, and saves its contents in the user DB.
func LoadCEX(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := GetSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("LoadDB", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(res, req)
	}

	vars := mux.Vars(req)
	cex := vars["cex"]                               //get the name of the CEX file from URL
	http_req := config.Host + "/cex/" + cex + ".cex" //build the URL to pass to cexHandler
	data, _ := getContent(http_req)                  //get response data using getContent and cexHandler
	str := string(data)                              //make the response data a string
	var urns, areas []string
	var catalog []BoltCatalog

	//read in the relations of the CEX file cutting away all unnecessary signs
	if strings.Contains(str, "#!relations") {
		relations := strings.Split(str, "#!relations")[1]
		relations = strings.Split(relations, "#!")[0]
		re := regexp.MustCompile("(?m)[\r\n]*^//.*$")
		relations = re.ReplaceAllString(relations, "")

		reader := csv.NewReader(strings.NewReader(relations))
		reader.Comma = '#'
		reader.LazyQuotes = true
		reader.FieldsPerRecord = 3

		for {
			line, error := reader.Read()
			if error == io.EOF {
				break
			} else if error != nil {
				log.Fatal(error)
			}
			if strings.Contains(line[1], "appearsOn") {
				urns = append(urns, line[0])
				areas = append(areas, line[2])
			}
		}
	}

	if strings.Contains(str, "#!ctscatalog") {
		ctsCatalog := strings.Split(str, "#!ctscatalog")[1]
		ctsCatalog = strings.Split(ctsCatalog, "#!")[0]
		re := regexp.MustCompile("(?m)[\r\n]*^//.*$")
		ctsCatalog = re.ReplaceAllString(ctsCatalog, "")

		var caturns, catcits, catgrps, catwrks, catvers, catexpls, onlines, languages []string
		// var languages [][]string

		reader := csv.NewReader(strings.NewReader(ctsCatalog))
		reader.Comma = '#'
		reader.LazyQuotes = true
		reader.FieldsPerRecord = -1
		reader.TrimLeadingSpace = true

		for {
			line, error := reader.Read()
			if error == io.EOF {
				break
			} else if error != nil {
				log.Fatal(error)
			}

			switch {
			case len(line) == 8:
				if line[0] != "urn" {
					caturns = append(caturns, line[0])
					catcits = append(catcits, line[1])
					catgrps = append(catgrps, line[2])
					catwrks = append(catwrks, line[3])
					catvers = append(catvers, line[4])
					catexpls = append(catexpls, line[5])
					onlines = append(onlines, line[6])
					languages = append(languages, line[7])
				}
			case len(line) != 8:
				fmt.Println("Catalogue Data not well formatted")
			}
		}
		for j := range caturns {
			catalog = append(catalog, BoltCatalog{URN: caturns[j], Citation: catcits[j], GroupName: catgrps[j], WorkTitle: catwrks[j], VersionLabel: catvers[j], ExemplarLabel: catexpls[j], Online: onlines[j], Language: languages[j]})
		}
	}

	ctsdata := strings.Split(str, "#!ctsdata")[1]
	ctsdata = strings.Split(ctsdata, "#!")[0]
	re := regexp.MustCompile("(?m)[\r\n]*^//.*$")
	ctsdata = re.ReplaceAllString(ctsdata, "")

	reader := csv.NewReader(strings.NewReader(ctsdata))
	reader.Comma = '#'
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	var texturns, text []string

	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			fmt.Println(line)
			log.Fatal(error)
		}
		switch {
		case len(line) == 2:
			texturns = append(texturns, line[0])
			text = append(text, line[1])
		case len(line) > 2:
			texturns = append(texturns, line[0])
			var textstring string
			for j := 1; j < len(line); j++ {
				textstring = textstring + line[j]
			}
			text = append(text, textstring)
		case len(line) < 2:
			fmt.Println("Wrong line:", line)
		}
	}

	works := append([]string(nil), texturns...)
	for i := range texturns {
		works[i] = strings.Join(strings.Split(texturns[i], ":")[0:4], ":") + ":"
	}
	works = removeDuplicatesUnordered(works)
	var boltworks []BoltWork
	var sortedcatalog []BoltCatalog
	for i := range works {
		work := works[i]
		testexist := false
		for j := range catalog {
			if catalog[j].URN == work {
				sortedcatalog = append(sortedcatalog, catalog[j])
				testexist = true
			}
		}
		if testexist == false {
			fmt.Println(works[i], " has not catalog entry")
			sortedcatalog = append(sortedcatalog, BoltCatalog{})
		}

		var bolturns []BoltURN
		var boltkeys []string
		for j := range texturns {
			if strings.Contains(texturns[j], work) {
				var textareas []string
				if contains(urns, texturns[j]) {
					for k := range urns {
						if urns[k] == texturns[j] {
							textareas = append(textareas, areas[k])
						}
					}
				}
				linetext := strings.Split(text[j], "-NEWLINE-")
				bolturns = append(bolturns, BoltURN{URN: texturns[j], Text: text[j], LineText: linetext, ImageRef: textareas})
				boltkeys = append(boltkeys, texturns[j])
			}
		}
		for j := range bolturns {
			bolturns[j].Index = j + 1
			switch {
			case j+1 == len(bolturns):
				bolturns[j].Next = ""
			default:
				bolturns[j].Next = bolturns[j+1].URN
			}
			switch {
			case j == 0:
				bolturns[j].Previous = ""
			default:
				bolturns[j].Previous = bolturns[j-1].URN
			}
			bolturns[j].Last = bolturns[len(bolturns)-1].URN
			bolturns[j].First = bolturns[0].URN
		}
		boltworks = append(boltworks, BoltWork{Key: boltkeys, Data: bolturns})
	}
	boltdata := BoltData{Bucket: works, Data: boltworks, Catalog: sortedcatalog}

	// write to database
	pwd, _ := os.Getwd()
	dbname := pwd + "/" + user + ".db"
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	for i := range boltdata.Bucket {
		newbucket := boltdata.Bucket[i]
		/// new stuff
		newcatkey := boltdata.Bucket[i]
		newcatnode, _ := json.Marshal(boltdata.Catalog[i])
		catkey := []byte(newcatkey)
		catvalue := []byte(newcatnode)
		err = db.Update(func(tx *bolt.Tx) error {
			bucket, err := tx.CreateBucketIfNotExists([]byte(newbucket))
			if err != nil {
				return err
			}

			err = bucket.Put(catkey, catvalue)
			if err != nil {
				return err
			}
			return nil
		})

		if err != nil {
			log.Fatal(err)
		}
		/// end stuff

		for j := range boltdata.Data[i].Key {
			newkey := boltdata.Data[i].Key[j]
			newnode, _ := json.Marshal(boltdata.Data[i].Data[j])
			key := []byte(newkey)
			value := []byte(newnode)
			// store some data
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

			if err != nil {
				log.Fatal(err)
			}
		}
	}
	io.WriteString(res, "Success")
	//This function should load a page using a template and display a propper success flash.
	//Alternatively it could become a helper function alltogether.
}

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
	session, err := GetSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("SaveImageRef", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(res, req)
	}

	vars := mux.Vars(req)
	newkey := vars["key"]
	imagerefstr := vars["updated"]
	fmt.Println("debug1", imagerefstr) //DEBUG
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	// imagerefstr := r.FormValue("text")
	imageref := strings.Split(imagerefstr, "+")
	fmt.Println("debug2", imageref) //DEBUG
	dbname := user + ".db"
	retrieveddata := BoltRetrieve(dbname, newbucket, newkey)
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	fmt.Println(retrievedjson.ImageRef) //DEBUG
	retrievedjson.ImageRef = imageref
	fmt.Println(imageref) //DEBUG
	newnode, _ := json.Marshal(retrievedjson)
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
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
	http.Redirect(res, req, "/view/"+newkey, http.StatusFound)
}

func SaveTranscription(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := GetSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("SaveTranscription", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(res, req)
	}

	vars := mux.Vars(req)
	newkey := vars["key"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	text := req.FormValue("text")
	linetext := strings.Split(text, "\r\n")
	text = strings.Replace(text, "\r\n", "", -1)
	dbname := user + ".db"
	retrieveddata := BoltRetrieve(dbname, newbucket, newkey)
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	retrievedjson.Text = text
	retrievedjson.LineText = linetext
	newnode, _ := json.Marshal(retrievedjson)
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
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
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
	http.Redirect(res, req, "/view/"+newkey, http.StatusFound)
}

func ExportCEX(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := GetSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := TestLoginStatus("ExportCEX", session)
	if loggedin {
		fmt.Println(message)
	} else {
		fmt.Println(message)
		Logout(res, req)
	}

	var texturns, texts, areas, imageurns []string
	var indexs []int
	vars := mux.Vars(req)
	filename := vars["filename"]
	dbname := user + ".db"
	buckets := Buckets(dbname)
	db, err := OpenBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		fmt.Printf("Error opening userDB: %s", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	for i := range buckets {
		db.View(func(tx *bolt.Tx) error {
			// Assume bucket exists and has keys
			b := tx.Bucket([]byte(buckets[i]))

			c := b.Cursor()

			for k, v := c.First(); k != nil; k, v = c.Next() {
				retrievedjson := BoltURN{}
				json.Unmarshal([]byte(v), &retrievedjson)
				ctsurn := retrievedjson.URN
				text := retrievedjson.Text
				index := retrievedjson.Index
				imageref := retrievedjson.ImageRef
				if len(imageref) > 0 {
					for i := range imageref {
						areas = append(areas, imageref[i])
						imageurns = append(imageurns, ctsurn)
					}
				}
				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				indexs = append(indexs, index)
			}

			return nil
		})
	}
	var correctedIndex []int
	k := 0
	for i := range indexs {
		if indexs[i] == 1 {
			k = i
		}
		result := k + indexs[i]
		correctedIndex = append(correctedIndex, result)
	}
	sort.Sort(dataframe{Indices: correctedIndex, Values1: texturns, Values2: texts})
	var content string
	content = "#!ctsdata\n"
	for i := range texturns {
		str := texturns[i] + "#" + texts[i] + "\n"
		content = content + str
	}
	content = content + "\n#!relations\n"
	for i := range imageurns {
		str := imageurns[i] + "#urn:cite2:dse:verbs.v1:appearsOn:#" + areas[i] + "\n"
		content = content + str
	}
	content = content + "\n"
	contentdispo := "Attachment; filename=" + filename + ".cex"
	modtime := time.Now()
	res.Header().Add("Content-Type", "text/plain; charset=utf-8")
	res.Header().Add("Content-Disposition", contentdispo)
	http.ServeContent(res, req, filename, modtime, bytes.NewReader([]byte(content)))
}
