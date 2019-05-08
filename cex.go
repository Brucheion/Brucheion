package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
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

// dataframe is the sort-matrix interface used in ExportCEX to sort integer Indices
//and their string values using used by sort.Sort in ExportCEX
type dataframe struct {
	Indices []int
	Values1 []string
	Values2 []string
}

//Len returns the number of Indices in a dataframe
func (m dataframe) Len() int { return len(m.Indices) }

//Less returns wether the in value of a dataframe index is smaller than the following index
func (m dataframe) Less(i, j int) bool { return m.Indices[i] < m.Indices[j] }

//Swap swaps the Indices and corresponding values withtin a dataframe
func (m dataframe) Swap(i, j int) {
	m.Indices[i], m.Indices[j] = m.Indices[j], m.Indices[i]
	m.Values1[i], m.Values1[j] = m.Values1[j], m.Values1[i]
	m.Values2[i], m.Values2[j] = m.Values2[j], m.Values2[i]
}

// ExportCEX exports CEX data from the user database to a CEX file
//Reference on CEX files: https://cite-architecture.github.io/citedx/CEX-spec-3.0.1/
func ExportCEX(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("ExportCEX", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	var texturns, texts, areas, imageurns []string
	var catalog []BoltCatalog
	var indexs []int
	vars := mux.Vars(req)
	filename := vars["filename"]
	dbname := user + ".db"
	buckets := Buckets(dbname)
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		log.Println(fmt.Errorf("ExportCEX: Error opening userDB for reading: %s", err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	for i := range buckets {
		db.View(func(tx *bolt.Tx) error {
			// Assume bucket exists and has keys
			bucket := tx.Bucket([]byte(buckets[i]))

			cursor := bucket.Cursor()

			for key, value := cursor.First(); key != nil; key, value = cursor.Next() {
				retrievedcatalog := BoltCatalog{}
				json.Unmarshal([]byte(value), &retrievedcatalog)
				catalog = append(catalog, retrievedcatalog)
			}

			for key, value := cursor.First(); key != nil; key, value = cursor.Next() {

				retrievedjson := gocite.Passage{}
				json.Unmarshal([]byte(value), &retrievedjson)
				ctsurn := retrievedjson.PassageID
				text := retrievedjson.Text.TXT
				index := retrievedjson.Index
				//imageref := retrievedjson.ImageRef
				imageref := []string{}
				for _, tmp := range retrievedjson.ImageLinks {
					imageref = append(imageref, tmp.Object)
				}
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

	content = "#!ctscatalog\nurn#citationScheme#groupName#workTitle#versionLabel#exemplarLabel#online#lang\n"
	for i := range catalog {
		str := catalog[i].URN + "#" + catalog[i].Citation + "#" + catalog[i].GroupName + "#" + catalog[i].WorkTitle + "#" + catalog[i].VersionLabel + "#" + catalog[i].ExemplarLabel + "#" + catalog[i].Online + "#" + catalog[i].Language + "\n"
		if str != "#######\n" {
			content = content + str
		}
	}
	content = content + "\n\n#!ctsdata\n"

	//content = "#!ctsdata\n"
	for i := range texturns {
		str := texturns[i] + "#" + texts[i] + "\n"
		if str != "#\n" {
			content = content + str
		}
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

//LoadCEX loads a CEX file, parses it, and saves its contents in the user DB.
//Maybe pass the parsed content to function in db.go?
func LoadCEX(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("LoadCEX", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	cex := vars["cex"]                              //get the name of the CEX file from URL
	httpReq := config.Host + "/cex/" + cex + ".cex" //build the URL to pass to cexHandler
	data, err := getContent(httpReq)                //get response data using getContent and cexHandler
	if err != nil {
		message = ("CEX file " + cex + ".cex not found.")
		log.Println(message)
		io.WriteString(res, message)
		return
	}

	str := string(data) //make the response data a string
	var urns, areas []string
	var catalog []BoltCatalog

	message = "LoadCEX: loading " + cex + ".cex"
	log.Println(message)
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

	//read in the ctscatalog (if exists)
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
				log.Println("Catalogue Data not well formatted")
			}
		}
		for j := range caturns {
			catalog = append(catalog, BoltCatalog{URN: caturns[j], Citation: catcits[j], GroupName: catgrps[j], WorkTitle: catwrks[j], VersionLabel: catvers[j], ExemplarLabel: catexpls[j], Online: onlines[j], Language: languages[j]})
		}
	}

	//read in the cts data
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
	var boltworks []gocite.Work
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
			log.Println(fmt.Println(works[i], " has no catalog entry"))
			sortedcatalog = append(sortedcatalog, BoltCatalog{})
		}

		var bolturns []gocite.Passage
		for j := range texturns {
			if strings.Contains(texturns[j], work) {
				var textareas []gocite.Triple
				if contains(urns, texturns[j]) {
					for k := range urns {
						if urns[k] == texturns[j] {
							textareas = append(textareas, gocite.Triple{Subject: texturns[j],
								Verb:   "urn:cite2:dse:verbs.v1:appears_on",
								Object: areas[k]})
						}
					}
				}
				linetext := strings.Replace(text[j], "-NEWLINE-", "\r\n", -1)
				bolturns = append(bolturns, gocite.Passage{PassageID: texturns[j],
					Range: false,
					Text: gocite.EncText{Brucheion: text[j],
						TXT: linetext},
					ImageLinks: textareas})
			}
		}
		for j := range bolturns {
			bolturns[j].Index = j + 1
			switch {
			case j+1 == len(bolturns):
				bolturns[j].Next = gocite.PassLoc{Exists: false}
			default:
				bolturns[j].Next = gocite.PassLoc{Exists: true, PassageID: bolturns[j+1].PassageID, Index: j + 1}
			}
			switch {
			case j == 0:
				bolturns[j].Prev = gocite.PassLoc{Exists: false}
			default:
				bolturns[j].Prev = gocite.PassLoc{Exists: true, PassageID: bolturns[j-1].PassageID, Index: j - 1}
			}
			bolturns[j].Last = gocite.PassLoc{Exists: true, PassageID: bolturns[len(bolturns)-1].PassageID, Index: len(bolturns) - 1}
			bolturns[j].First = gocite.PassLoc{Exists: true, PassageID: bolturns[0].PassageID, Index: 0}
		}
		boltworks = append(boltworks, gocite.Work{WorkID: work, Passages: bolturns, Ordered: true})
	}
	boltdata := BoltData{Bucket: works, Data: boltworks, Catalog: sortedcatalog}

	// write to database
	pwd, _ := os.Getwd()
	dbname := pwd + "/" + user + ".db"
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		log.Println(fmt.Printf("LoadCEX: error opening userDB for writing: %s", err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	for i := range boltdata.Bucket {
		newbucket := boltdata.Bucket[i]
		/// new stuff
		//Saving the CTS Catalog data
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

		//saving the individual passages
		for j := range boltdata.Data[i].Passages {
			newkey := boltdata.Data[i].Passages[j].PassageID
			newnode, _ := json.Marshal(boltdata.Data[i].Passages[j])
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
	message = ("CEX file " + cex + ".cex loaded into Brucheion successfully.")
	log.Println(message)
	io.WriteString(res, message)
	//This function should load a page using a template and display a propper success flash.
	//Alternatively it could become a helper function alltogether.
}
