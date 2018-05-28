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
	"strconv"
	"strings"
	"time"

	"github.com/aebruno/nwalgo"
	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
)

type Transcription struct {
	CTSURN        string
	Transcriber   string
	Transcription string
	Previous      string
	Next          string
	First         string
	Last          string
	ImageRef      []string
	TextRef       []string
	ImageJS       string
	CatID         string
	CatCit        string
	CatGroup      string
	CatWork       string
	CatVers       string
	CatExmpl      string
	CatOn         string
	CatLan        string
}

type CompPage struct {
	User      string
	Title     string
	Text      template.HTML
	Port      string
	CatID     string
	CatCit    string
	CatGroup  string
	CatWork   string
	CatVers   string
	CatExmpl  string
	CatOn     string
	CatLan    string
	User2     string
	Title2    string
	Text2     template.HTML
	CatID2    string
	CatCit2   string
	CatGroup2 string
	CatWork2  string
	CatVers2  string
	CatExmpl2 string
	CatOn2    string
	CatLan2   string
}

type Page struct {
	User         string
	Title        string
	ImageJS      string
	ImageScript  template.HTML
	ImageHTML    template.HTML
	TextHTML     template.HTML
	Text         template.HTML
	Previous     string
	Next         string
	PreviousLink template.HTML
	NextLink     template.HTML
	First        string
	Last         string
	Port         string
	ImageRef     string
	CatID        string
	CatCit       string
	CatGroup     string
	CatWork      string
	CatVers      string
	CatExmpl     string
	CatOn        string
	CatLan       string
}

var templates = template.Must(template.ParseFiles("tmpl/view.html", "tmpl/edit.html", "tmpl/edit2.html", "tmpl/editcat.html", "tmpl/compare.html", "tmpl/consolidate.html", "tmpl/tree.html", "tmpl/crud.html"))
var jstemplates = template.Must(template.ParseFiles("js/ict2.js"))

func main() {
	serverIP := ":7000"
	router := mux.NewRouter().StrictSlash(true)
	s := http.StripPrefix("/static/", http.FileServer(http.Dir("./static/")))
	js := http.StripPrefix("/js/", http.FileServer(http.Dir("./js/")))
	cex := http.StripPrefix("/cex/", http.FileServer(http.Dir("./cex/")))
	router.PathPrefix("/static/").Handler(s)
	router.PathPrefix("/js/").Handler(js)
	router.PathPrefix("/cex/").Handler(cex)
	router.HandleFunc("/{user}/{urn}/treenode.json", Treenode)
	router.HandleFunc("/{user}/main/", MainPage)
	router.HandleFunc("/{user}/load/{cex}", LoadDB)
	router.HandleFunc("/{user}/new/{key}", newText)
	router.HandleFunc("/{user}/view/{urn}", ViewPage)
	router.HandleFunc("/{user}/tree/", TreePage)
	router.HandleFunc("/{user}/edit/{urn}", EditPage)
	router.HandleFunc("/{user}/editcat/{urn}", EditCatPage)
	router.HandleFunc("/{user}/save/{key}", SaveTranscription)
	router.HandleFunc("/{user}/addNodeAfter/{key}", AddNodeAfter)
	router.HandleFunc("/{user}/addFirstNode/{key}", AddFirstNode)
	router.HandleFunc("/{user}/crud/", CrudPage)
	router.HandleFunc("/{user}/deleteBucket/{urn}", deleteBucket)
	router.HandleFunc("/{user}/deleteNode/{urn}", deleteNode)
	router.HandleFunc("/{user}/export/{filename}", ExportCEX)
	router.HandleFunc("/{user}/edit2/{urn}", Edit2Page)
	router.HandleFunc("/{user}/compare/{urn}+{urn2}", comparePage)
	router.HandleFunc("/{user}/consolidate/{urn}+{urn2}", consolidatePage)
	router.HandleFunc("/{user}/saveImage/{key}", SaveImageRef)
	log.Println("Listening at" + serverIP + "...")
	log.Fatal(http.ListenAndServe(serverIP, router))
}

func MainPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	dbname := user + ".db"
	buckets := Buckets(dbname)
	fmt.Println(buckets)
}

func TreePage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	dbname := user + ".db"

	textref := Buckets(dbname)

	transcription := Transcription{
		Transcriber: user,
		TextRef:     textref}
	port := ":7000"
	p, _ := loadCrudPage(transcription, port)
	renderTemplate(w, "tree", p)
}

func CrudPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	dbname := user + ".db"

	textref := Buckets(dbname)

	transcription := Transcription{
		Transcriber: user,
		TextRef:     textref}
	port := ":7000"
	p, _ := loadCrudPage(transcription, port)
	renderTemplate(w, "crud", p)
}

func loadCrudPage(transcription Transcription, port string) (*Page, error) {
	user := transcription.Transcriber
	var textrefrences []string
	for i := range transcription.TextRef {
		textrefrences = append(textrefrences, transcription.TextRef[i])
	}
	textref := strings.Join(textrefrences, " ")
	return &Page{User: user, Text: template.HTML(textref), Port: port}, nil
}

func ExportCEX(w http.ResponseWriter, r *http.Request) {
	var texturns, texts, areas, imageurns []string
	var indexs []int
	vars := mux.Vars(r)
	filename := vars["filename"]
	user := vars["user"]
	dbname := user + ".db"
	buckets := Buckets(dbname)
	db, err := bolt.Open(dbname, 0644, nil)
	if err != nil {
		log.Fatal(err)
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
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.Header().Add("Content-Disposition", contentdispo)
	http.ServeContent(w, r, filename, modtime, bytes.NewReader([]byte(content)))
}

func SaveImageRef(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	newkey := vars["key"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	user := vars["user"]
	imagerefstr := r.FormValue("text")
	imageref := strings.Split(imagerefstr, "#")
	dbname := user + ".db"
	retrieveddata := BoltRetrieve(dbname, newbucket, newkey)
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	retrievedjson.ImageRef = imageref
	newnode, _ := json.Marshal(retrievedjson)
	db, err := bolt.Open(dbname, 0644, nil)
	if err != nil {
		log.Fatal(err)
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
	http.Redirect(w, r, "/"+user+"/view/"+newkey, http.StatusFound)
}

func AddFirstNode(w http.ResponseWriter, r *http.Request) {
	var texturns, texts, previouss, nexts, firsts, lasts []string
	var imagerefs, linetexts [][]string
	var indexs []int
	vars := mux.Vars(r)
	newkey := vars["key"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	user := vars["user"]

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
	db, err := bolt.Open(dbname, 0644, nil)
	if err != nil {
		log.Fatal(err)
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

	var bolturns []BoltURN
	for i := range texturns {
		bolturns = append(bolturns, BoltURN{URN: texturns[i],
			Text:     texts[i],
			LineText: linetexts[i],
			Previous: previouss[i],
			Next:     nexts[i],
			First:    firsts[i],
			Last:     lasts[i],
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

func AddNodeAfter(w http.ResponseWriter, r *http.Request) {
	var texturns, texts, previouss, nexts, firsts, lasts []string
	var imagerefs, linetexts [][]string
	var indexs []int
	vars := mux.Vars(r)
	newkey := vars["key"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	user := vars["user"]

	dbname := user + ".db"
	retrieveddata := BoltRetrieve(dbname, newbucket, newkey)
	retrievednodejson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievednodejson)
	bookmark := retrievednodejson.Index
	lastnode := false
	if retrievednodejson.Last == retrievednodejson.URN {
		lastnode = true
	}
	db, err := bolt.Open(dbname, 0644, nil)
	if err != nil {
		log.Fatal(err)
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
			first := retrievedjson.First
			imageref := retrievedjson.ImageRef
			last := retrievedjson.Last
			index := retrievedjson.Index

			switch {
			case index < bookmark:
				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				linetexts = append(linetexts, linetext)
				previouss = append(previouss, previous)
				nexts = append(nexts, next)
				firsts = append(firsts, first)
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
				firsts = append(firsts, first)
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

				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				linetexts = append(linetexts, linetext)
				previouss = append(previouss, previous)
				nexts = append(nexts, newnode)
				firsts = append(firsts, first)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, index)
				imagerefs = append(imagerefs, imageref)

				texturns = append(texturns, newnode)
				texts = append(texts, "")
				linetexts = append(linetexts, []string{})
				previouss = append(previouss, ctsurn)
				nexts = append(nexts, next)
				firsts = append(firsts, first)
				switch lastnode {
				case false:
					lasts = append(lasts, last)
				case true:
					newlast := newbucket + "newNode" + strconv.Itoa(bookmark)
					lasts = append(lasts, newlast)
				}
				indexs = append(indexs, newindex)
				imagerefs = append(imagerefs, []string{})
			case index == bookmark+1:
				newnode := newbucket + "newNode" + strconv.Itoa(bookmark)
				newindex := index + 1
				texturns = append(texturns, ctsurn)
				texts = append(texts, text)
				linetexts = append(linetexts, linetext)
				previouss = append(previouss, newnode)
				nexts = append(nexts, next)
				firsts = append(firsts, first)
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

	var bolturns []BoltURN
	for i := range texturns {
		bolturns = append(bolturns, BoltURN{URN: texturns[i],
			Text:     texts[i],
			LineText: linetexts[i],
			Previous: previouss[i],
			Next:     nexts[i],
			First:    firsts[i],
			Last:     lasts[i],
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

func newText(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	newkey := vars["key"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	user := vars["user"]
	dbname := user + ".db"
	retrievedjson := BoltURN{}
	retrievedjson.URN = newkey
	newnode, _ := json.Marshal(retrievedjson)
	db, err := bolt.Open(dbname, 0644, nil)
	if err != nil {
		log.Fatal(err)
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
	http.Redirect(w, r, "/"+user+"/view/"+newkey, http.StatusFound)
}

func SaveTranscription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	newkey := vars["key"]
	newbucket := strings.Join(strings.Split(newkey, ":")[0:4], ":") + ":"
	user := vars["user"]
	text := r.FormValue("text")
	linetext := strings.Split(text, "\r\n")
	text = strings.Replace(text, "\r\n", "", -1)
	dbname := user + ".db"
	retrieveddata := BoltRetrieve(dbname, newbucket, newkey)
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	retrievedjson.Text = text
	retrievedjson.LineText = linetext
	newnode, _ := json.Marshal(retrievedjson)
	db, err := bolt.Open(dbname, 0644, nil)
	if err != nil {
		log.Fatal(err)
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
	http.Redirect(w, r, "/"+user+"/view/"+newkey, http.StatusFound)
}

func LoadDB(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cex := vars["cex"]
	user := vars["user"]
	http_req := "http://localhost:7000/cex/" + cex + ".cex"
	data, _ := getContent(http_req)
	str := string(data)
	var urns, areas []string
	var catalog []BoltCatalog

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
	db, err := bolt.Open(dbname, 0644, nil)
	if err != nil {
		log.Fatal(err)
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
	io.WriteString(w, "Success")
}

func loadPage(transcription Transcription, port string) (*Page, error) {
	user := transcription.Transcriber
	imagejs := transcription.ImageJS
	title := transcription.CTSURN
	text := transcription.Transcription
	previous := transcription.Previous
	next := transcription.Next
	first := transcription.First
	last := transcription.Last
	catid := transcription.CatID
	catcit := transcription.CatCit
	catgroup := transcription.CatGroup
	catwork := transcription.CatWork
	catversion := transcription.CatVers
	catexpl := transcription.CatExmpl
	caton := transcription.CatOn
	catlan := transcription.CatLan

	dbname := user + ".db"
	var previouslink, nextlink string
	switch {
	case previous == "":
		previouslink = `<a href ="/` + user + `/new/">add previous</a>`
		previous = title
	default:
		previouslink = `<a href ="/` + user + `/view/` + previous + `">` + previous + `</a>`
	}
	switch {
	case next == "":
		nextlink = `<a href ="/` + user + `/new/">add next</a>`
		next = title
	default:
		nextlink = `<a href ="/` + user + `/view/` + next + `">` + next + `</a>`
	}
	var textrefrences []string
	for i := range transcription.TextRef {
		requestedbucket := transcription.TextRef[i]
		texturn := requestedbucket + strings.Split(title, ":")[4]

		// adding testing if requestedbucket exists...
		retrieveddata := BoltRetrieve(dbname, requestedbucket, texturn)
		retrievedjson := BoltURN{}
		json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)

		ctsurn := retrievedjson.URN
		var htmllink string
		switch {
		case ctsurn == title:
			htmllink = `<option value="/` + user + "/view/" + ctsurn + `" selected>` + transcription.TextRef[i] + `</option>`
		case ctsurn == "":
			ctsurn = BoltRetrieveFirstKey(dbname, requestedbucket)
			htmllink = `<option value="/` + user + "/view/" + ctsurn + `">` + transcription.TextRef[i] + `</option>`
		default:
			htmllink = `<option value="/` + user + "/view/" + ctsurn + `">` + transcription.TextRef[i] + `</option>`
		}
		textrefrences = append(textrefrences, htmllink)
	}
	textref := strings.Join(textrefrences, " ")
	imageref := strings.Join(transcription.ImageRef, "#")
	beginjs := `<script type="text/javascript">
	window.onload = function() {`
	startjs := `
		var a`
	start2js := `= document.getElementById("imageLink`
	middlejs := `");
	a`
	middle2js := `.onclick = function() {
		imgUrn="`
	endjs := `"
	reloadImage();
	return false;
}`
	finaljs := `
}
</script>`
	starthtml := `<a id="imageLink`
	middlehtml := `">`
	endhtml := ` </a>`
	var jsstrings, htmlstrings []string
	jsstrings = append(jsstrings, beginjs)
	for i := range transcription.ImageRef {
		jsstring := startjs + strconv.Itoa(i) + start2js + strconv.Itoa(i) + middlejs + strconv.Itoa(i) + middle2js + transcription.ImageRef[i] + endjs
		jsstrings = append(jsstrings, jsstring)
		htmlstring := starthtml + strconv.Itoa(i) + middlehtml + transcription.ImageRef[i] + endhtml
		htmlstrings = append(htmlstrings, htmlstring)
	}
	jsstrings = append(jsstrings, finaljs)
	jsstring := strings.Join(jsstrings, "")
	htmlstring := strings.Join(htmlstrings, "")
	imagescript := template.HTML(jsstring)
	imagehtml := template.HTML(htmlstring)
	texthtml := template.HTML(textref)
	previoushtml := template.HTML(previouslink)
	nexthtml := template.HTML(nextlink)
	return &Page{User: user,
		Title:        title,
		Text:         template.HTML(text),
		Previous:     previous,
		PreviousLink: previoushtml,
		Next:         next,
		NextLink:     nexthtml,
		First:        first,
		Last:         last,
		ImageScript:  imagescript,
		ImageHTML:    imagehtml,
		TextHTML:     texthtml,
		ImageRef:     imageref,
		CatID:        catid,
		CatCit:       catcit,
		CatGroup:     catgroup,
		CatWork:      catwork,
		CatVers:      catversion,
		CatExmpl:     catexpl,
		CatOn:        caton,
		CatLan:       catlan,
		Port:         port,
		ImageJS:      imagejs}, nil
}

func loadCompPage(transcription, transcription2 Transcription, port string) (*CompPage, error) {
	user := transcription.Transcriber
	title := transcription.CTSURN
	text := transcription.Transcription
	catid := transcription.CatID
	catcit := transcription.CatCit
	catgroup := transcription.CatGroup
	catwork := transcription.CatWork
	catversion := transcription.CatVers
	catexpl := transcription.CatExmpl
	caton := transcription.CatOn
	catlan := transcription.CatLan

	title2 := transcription2.CTSURN
	text2 := transcription2.Transcription
	catid2 := transcription2.CatID
	catcit2 := transcription2.CatCit
	catgroup2 := transcription2.CatGroup
	catwork2 := transcription2.CatWork
	catversion2 := transcription2.CatVers
	catexpl2 := transcription2.CatExmpl
	caton2 := transcription2.CatOn
	catlan2 := transcription2.CatLan

	texts := nwa(text, text2)

	return &CompPage{User: user,
		Title:     title,
		Text:      template.HTML(texts[0]),
		CatID:     catid,
		CatCit:    catcit,
		CatGroup:  catgroup,
		CatWork:   catwork,
		CatVers:   catversion,
		CatExmpl:  catexpl,
		CatOn:     caton,
		CatLan:    catlan,
		Title2:    title2,
		Text2:     template.HTML(texts[1]),
		CatID2:    catid2,
		CatCit2:   catcit2,
		CatGroup2: catgroup2,
		CatWork2:  catwork2,
		CatVers2:  catversion2,
		CatExmpl2: catexpl2,
		CatOn2:    caton2,
		CatLan2:   catlan2,
		Port:      port}, nil
}

func nwa(text, text2 string) []string {
	start := `<div class="tile is-child" lnum="L1">`
	start2 := `<div class="tile is-child" lnum="L2">`
	end := `</div>`
	collection := []string{text, text2}
	for i := range collection {
		collection[i] = strings.ToLower(collection[i])
	}
	basestring := strings.Fields(collection[0])
	var basetext []Word
	var comparetext []Word
	for i := range basestring {
		basetext = append(basetext, Word{Appearance: basestring[i], Id: i + 1})
	}

	comparestring := strings.Fields(collection[1])
	comparetext = []Word{}
	for i := range comparestring {
		comparetext = append(comparetext, Word{Appearance: comparestring[i]})
	}
	aln1, aln2, score := nwalgo.Align(collection[0], collection[1], 1, -1, -1)
	aligned1 := strings.Fields(aln1)
	aligned2 := strings.Fields(aln2)
	var score_range []float64
	var index int
	for j := range aligned1 {
		score_range = []float64{}
		for i, _ := range aligned2 {
			aln1, aln2, score = nwalgo.Align(aligned1[j], aligned2[i], 1, -1, -1)
			var penalty float64
			switch {
			case i > j:
				penalty = float64((i - j)) / 2.0
			case i < j:
				penalty = float64((j - i)) / 2.0
			default:
				penalty = 0
			}
			var f float64 = (float64(score) - penalty) / float64(len(aln1))
			score_range = append(score_range, f)
		}
		index = maxfloat(score_range)
		comparetext[index].Alignment = j + 1
		comparetext[index].Id = index + 1
		basetext[j].Id = j + 1
	}
	text2 = start2
	for i := range comparetext {
		switch comparetext[i].Id {
		case 0:
			text2 = text2 + "<w id=\"" + strconv.Itoa(i+1) + "\" alignment=\"" + strconv.Itoa(comparetext[i].Alignment) + "\">" + comparetext[i].Appearance + "</w>" + " "
		default:
			text2 = text2 + "<w id=\"" + strconv.Itoa(i+1) + "\" alignment=\"" + strconv.Itoa(comparetext[i].Alignment) + "\">" + comparetext[i].Appearance + "</w>" + " "
		}
	}
	text2 = text2 + end

	text = start
	for i := range basetext {
		for j := range comparetext {
			if comparetext[j].Alignment == basetext[i].Id {
				basetext[i].Alignment = comparetext[j].Id
			}
		}
		text = text + "<w id=\"" + strconv.Itoa(basetext[i].Id) + "\" alignment=\"" + strconv.Itoa(basetext[i].Alignment) + "\">" + basetext[i].Appearance + "</w>" + " "
	}
	text = text + end

	return []string{text, text2}
}

func maxfloat(floatslice []float64) int {
	max := floatslice[0]
	maxindex := 0
	for i, value := range floatslice {
		if value > max {
			max = value
			maxindex = i
		}
	}
	return maxindex
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func renderCompTemplate(w http.ResponseWriter, tmpl string, p *CompPage) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ViewPage generates the webpage based on the sent request
func ViewPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	urn := vars["urn"]
	user := vars["user"]
	dbname := user + ".db"

	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedcat := BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson := BoltCatalog{}
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	ctsurn := retrievedjson.URN
	text := "<p>"
	linetext := retrievedjson.LineText
	for i := range linetext {
		text = text + linetext[i]
		if i < len(linetext)-1 {
			text = text + "<br>"
		}
	}
	text = text + "</p>"
	previous := retrievedjson.Previous
	next := retrievedjson.Next
	imageref := retrievedjson.ImageRef
	first := retrievedjson.First
	last := retrievedjson.Last
	imagejs := "urn:cite2:test:googleart.positive:DuererHare1502"
	switch len(imageref) > 0 {
	case true:
		imagejs = imageref[0]
	}
	catid := retrievedcatjson.URN
	catcit := retrievedcatjson.Citation
	catgroup := retrievedcatjson.GroupName
	catwork := retrievedcatjson.WorkTitle
	catversion := retrievedcatjson.VersionLabel
	catexpl := retrievedcatjson.ExemplarLabel
	caton := retrievedcatjson.Online
	catlan := retrievedcatjson.Language

	transcription := Transcription{CTSURN: ctsurn,
		Transcriber:   user,
		Transcription: text,
		Previous:      previous,
		Next:          next,
		First:         first,
		Last:          last,
		TextRef:       textref,
		ImageRef:      imageref,
		ImageJS:       imagejs,
		CatID:         catid,
		CatCit:        catcit,
		CatGroup:      catgroup,
		CatWork:       catwork,
		CatVers:       catversion,
		CatExmpl:      catexpl,
		CatOn:         caton,
		CatLan:        catlan}

	port := ":7000"
	p, _ := loadPage(transcription, port)
	renderTemplate(w, "view", p)
}

func comparePage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	urn := vars["urn"]
	urn2 := vars["urn2"]
	user := vars["user"]
	dbname := user + ".db"

	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedcat := BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson := BoltCatalog{}
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	ctsurn := retrievedjson.URN
	text := ""
	linetext := retrievedjson.LineText
	for i := range linetext {
		text = text + linetext[i]
		if i < len(linetext)-1 {
			text = text + " "
		}
	}
	previous := retrievedjson.Previous
	next := retrievedjson.Next
	imageref := retrievedjson.ImageRef
	first := retrievedjson.First
	last := retrievedjson.Last
	imagejs := "urn:cite2:test:googleart.positive:DuererHare1502"
	switch len(imageref) > 0 {
	case true:
		imagejs = imageref[0]
	}
	catid := retrievedcatjson.URN
	catcit := retrievedcatjson.Citation
	catgroup := retrievedcatjson.GroupName
	catwork := retrievedcatjson.WorkTitle
	catversion := retrievedcatjson.VersionLabel
	catexpl := retrievedcatjson.ExemplarLabel
	caton := retrievedcatjson.Online
	catlan := retrievedcatjson.Language

	transcription := Transcription{CTSURN: ctsurn,
		Transcriber:   user,
		Transcription: text,
		Previous:      previous,
		Next:          next,
		First:         first,
		Last:          last,
		TextRef:       textref,
		ImageRef:      imageref,
		ImageJS:       imagejs,
		CatID:         catid,
		CatCit:        catcit,
		CatGroup:      catgroup,
		CatWork:       catwork,
		CatVers:       catversion,
		CatExmpl:      catexpl,
		CatOn:         caton,
		CatLan:        catlan}

	requestedbucket = strings.Join(strings.Split(urn2, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata = BoltRetrieve(dbname, requestedbucket, urn2)
	retrievedcat = BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson = BoltCatalog{}
	retrievedjson = BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	ctsurn = retrievedjson.URN
	text = ""
	linetext = retrievedjson.LineText
	for i := range linetext {
		text = text + linetext[i]
		if i < len(linetext)-1 {
			text = text + " "
		}
	}
	previous = retrievedjson.Previous
	next = retrievedjson.Next
	imageref = retrievedjson.ImageRef
	first = retrievedjson.First
	last = retrievedjson.Last
	imagejs = "urn:cite2:test:googleart.positive:DuererHare1502"
	switch len(imageref) > 0 {
	case true:
		imagejs = imageref[0]
	}
	catid = retrievedcatjson.URN
	catcit = retrievedcatjson.Citation
	catgroup = retrievedcatjson.GroupName
	catwork = retrievedcatjson.WorkTitle
	catversion = retrievedcatjson.VersionLabel
	catexpl = retrievedcatjson.ExemplarLabel
	caton = retrievedcatjson.Online
	catlan = retrievedcatjson.Language

	transcription2 := Transcription{CTSURN: ctsurn,
		Transcriber:   user,
		Transcription: text,
		Previous:      previous,
		Next:          next,
		First:         first,
		Last:          last,
		TextRef:       textref,
		ImageRef:      imageref,
		ImageJS:       imagejs,
		CatID:         catid,
		CatCit:        catcit,
		CatGroup:      catgroup,
		CatWork:       catwork,
		CatVers:       catversion,
		CatExmpl:      catexpl,
		CatOn:         caton,
		CatLan:        catlan}

	port := ":7000"

	p, _ := loadCompPage(transcription, transcription2, port)
	renderCompTemplate(w, "compare", p)
}

func consolidatePage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	urn := vars["urn"]
	urn2 := vars["urn2"]
	user := vars["user"]
	dbname := user + ".db"

	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedcat := BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson := BoltCatalog{}
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	ctsurn := retrievedjson.URN
	text := ""
	linetext := retrievedjson.LineText
	for i := range linetext {
		text = text + linetext[i]
		if i < len(linetext)-1 {
			text = text + " "
		}
	}
	previous := retrievedjson.Previous
	next := retrievedjson.Next
	imageref := retrievedjson.ImageRef
	first := retrievedjson.First
	last := retrievedjson.Last
	imagejs := "urn:cite2:test:googleart.positive:DuererHare1502"
	switch len(imageref) > 0 {
	case true:
		imagejs = imageref[0]
	}
	catid := retrievedcatjson.URN
	catcit := retrievedcatjson.Citation
	catgroup := retrievedcatjson.GroupName
	catwork := retrievedcatjson.WorkTitle
	catversion := retrievedcatjson.VersionLabel
	catexpl := retrievedcatjson.ExemplarLabel
	caton := retrievedcatjson.Online
	catlan := retrievedcatjson.Language

	transcription := Transcription{CTSURN: ctsurn,
		Transcriber:   user,
		Transcription: text,
		Previous:      previous,
		Next:          next,
		First:         first,
		Last:          last,
		TextRef:       textref,
		ImageRef:      imageref,
		ImageJS:       imagejs,
		CatID:         catid,
		CatCit:        catcit,
		CatGroup:      catgroup,
		CatWork:       catwork,
		CatVers:       catversion,
		CatExmpl:      catexpl,
		CatOn:         caton,
		CatLan:        catlan}

	requestedbucket = strings.Join(strings.Split(urn2, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata = BoltRetrieve(dbname, requestedbucket, urn2)
	retrievedcat = BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson = BoltCatalog{}
	retrievedjson = BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	ctsurn = retrievedjson.URN
	text = ""
	linetext = retrievedjson.LineText
	for i := range linetext {
		text = text + linetext[i]
		if i < len(linetext)-1 {
			text = text + " "
		}
	}
	previous = retrievedjson.Previous
	next = retrievedjson.Next
	imageref = retrievedjson.ImageRef
	first = retrievedjson.First
	last = retrievedjson.Last
	imagejs = "urn:cite2:test:googleart.positive:DuererHare1502"
	switch len(imageref) > 0 {
	case true:
		imagejs = imageref[0]
	}
	catid = retrievedcatjson.URN
	catcit = retrievedcatjson.Citation
	catgroup = retrievedcatjson.GroupName
	catwork = retrievedcatjson.WorkTitle
	catversion = retrievedcatjson.VersionLabel
	catexpl = retrievedcatjson.ExemplarLabel
	caton = retrievedcatjson.Online
	catlan = retrievedcatjson.Language

	transcription2 := Transcription{CTSURN: ctsurn,
		Transcriber:   user,
		Transcription: text,
		Previous:      previous,
		Next:          next,
		First:         first,
		Last:          last,
		TextRef:       textref,
		ImageRef:      imageref,
		ImageJS:       imagejs,
		CatID:         catid,
		CatCit:        catcit,
		CatGroup:      catgroup,
		CatWork:       catwork,
		CatVers:       catversion,
		CatExmpl:      catexpl,
		CatOn:         caton,
		CatLan:        catlan}

	port := ":7000"

	p, _ := loadCompPage(transcription, transcription2, port)
	renderCompTemplate(w, "consolidate", p)
}

func EditCatPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	urn := vars["urn"]
	user := vars["user"]
	dbname := user + ".db"
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedcat := BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson := BoltCatalog{}
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	ctsurn := retrievedjson.URN
	catid := retrievedcatjson.URN
	catcit := retrievedcatjson.Citation
	catgroup := retrievedcatjson.GroupName
	catwork := retrievedcatjson.WorkTitle
	catversion := retrievedcatjson.VersionLabel
	catexpl := retrievedcatjson.ExemplarLabel
	caton := retrievedcatjson.Online
	catlan := retrievedcatjson.Language
	transcription := Transcription{CTSURN: ctsurn,
		Transcriber: user,
		CatID:       catid, CatCit: catcit, CatGroup: catgroup, CatWork: catwork, CatVers: catversion, CatExmpl: catexpl, CatOn: caton, CatLan: catlan}
	port := ":7000"
	p, _ := loadPage(transcription, port)
	renderTemplate(w, "editcat", p)
}

func EditPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	urn := vars["urn"]
	user := vars["user"]
	dbname := user + ".db"
	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)

	ctsurn := retrievedjson.URN
	linetext := retrievedjson.LineText
	previous := retrievedjson.Previous
	next := retrievedjson.Next
	imageref := retrievedjson.ImageRef
	first := retrievedjson.First
	last := retrievedjson.Last
	imagejs := "urn:cite2:test:googleart.positive:DuererHare1502"
	switch len(imageref) > 0 {
	case true:
		imagejs = imageref[0]
	}
	text := ""
	for i := range linetext {
		text = text + linetext[i]
		if i < len(linetext)-1 {
			text = text + "\r\n"
		}
	}
	transcription := Transcription{CTSURN: ctsurn,
		Transcriber:   user,
		Transcription: text,
		Previous:      previous,
		Next:          next,
		First:         first,
		Last:          last,
		TextRef:       textref,
		ImageRef:      imageref,
		ImageJS:       imagejs}
	port := ":7000"
	p, _ := loadPage(transcription, port)
	renderTemplate(w, "edit", p)
}

func Edit2Page(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	urn := vars["urn"]
	user := vars["user"]
	dbname := user + ".db"
	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)

	ctsurn := retrievedjson.URN
	text := retrievedjson.Text
	previous := retrievedjson.Previous
	next := retrievedjson.Next
	imageref := retrievedjson.ImageRef
	first := retrievedjson.First
	last := retrievedjson.Last
	imagejs := "urn:cite2:test:googleart.positive:DuererHare1502"
	switch len(imageref) > 0 {
	case true:
		imagejs = imageref[0]
	}
	transcription := Transcription{CTSURN: ctsurn,
		Transcriber:   user,
		Transcription: text,
		Previous:      previous,
		Next:          next,
		First:         first,
		Last:          last,
		TextRef:       textref,
		ImageRef:      imageref,
		ImageJS:       imagejs}
	port := ":7000"
	p, _ := loadPage(transcription, port)
	renderTemplate(w, "edit2", p)
}
