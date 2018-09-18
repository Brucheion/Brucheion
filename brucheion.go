package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
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

	"github.com/ThomasK81/gocite"
	"github.com/ThomasK81/gonwr"
	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
	"golang.org/x/net/html"
)

type JSONlist struct {
	Item []string `json:"item"`
}

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

var templates = template.Must(template.ParseFiles("tmpl/view.html", "tmpl/edit.html", "tmpl/edit2.html", "tmpl/editcat.html", "tmpl/compare.html", "tmpl/multicompare.html", "tmpl/consolidate.html", "tmpl/tree.html", "tmpl/crud.html"))
var jstemplates = template.Must(template.ParseFiles("js/ict2.js"))
var serverIP = ":7000"

func main() {
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
	router.HandleFunc("/{user}/multicompare/{urn}", MultiPage)
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
	router.HandleFunc("/{user}/newWork", newWork)
	router.HandleFunc("/{user}/newCollection/{name}/{urns}", newCollection)
	router.HandleFunc("/{user}/newCITECollection/{name}", newCITECollection)
	router.HandleFunc("/{user}/getImageInfo/{name}/{imageurn}", getImageInfo)
	router.HandleFunc("/{user}/addtoCITE", addCITE)

	router.HandleFunc("/{user}/requestImgID/{name}", requestImgID)
	router.HandleFunc("/{user}/deleteCollection", deleteCollection)
	router.HandleFunc("/{user}/requestImgCollection", requestImgCollection)
	log.Println("Listening at" + serverIP + "...")
	log.Fatal(http.ListenAndServe(serverIP, router))
}

// Helper function to pull the href attribute from a Token
func getHref(t html.Token) (ok bool, href string) {
	for _, a := range t.Attr {
		if a.Key == "href" {
			href = a.Val
			ok = true
		}
	}
	return
}

func extractLinks(urn gocite.Cite2Urn) (links []string, err error) {
	urnLink := urn.Namespace + "/" + strings.Replace(urn.Collection, ".", "/", -1) + "/"
	url := "http://localhost" + serverIP + "/static/image_archive/" + urnLink
	response, err := http.Get(url)
	if err != nil {
		return links, err
	}
	z := html.NewTokenizer(response.Body)
	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return
		case tt == html.StartTagToken:
			t := z.Token()

			isAnchor := t.Data == "a"
			if !isAnchor {
				continue
			}
			ok, url := getHref(t)
			if strings.Contains(url, ".dzi") {
				urnStr := urn.Base + ":" + urn.Protocol + ":" + urn.Namespace + ":" + urn.Collection + ":" + strings.Replace(url, ".dzi", "", -1)
				links = append(links, urnStr)
			}
			if !ok {
				continue
			}
		}
	}
	return links, nil
}

func requestImgCollection(w http.ResponseWriter, r *http.Request) {
	response := JSONlist{}
	vars := mux.Vars(r)
	user := vars["user"]
	dbname := user + ".db"
	db, err := bolt.Open(dbname, 0644, nil)
	if err != nil {
		log.Fatal(err)
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
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintln(w, string(resultJSON))
	}
	resultJSON, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(resultJSON))
}

func getImageInfo(w http.ResponseWriter, r *http.Request) {
	retImage := imageCollection{}
	newImage := image{}
	vars := mux.Vars(r)
	user := vars["user"]
	collectionName := vars["name"]
	imageurn := vars["imageurn"]
	dbkey := []byte(collectionName)
	dbname := user + ".db"
	db, err := bolt.Open(dbname, 0644, nil)
	if err != nil {
		log.Fatal(err)
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
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintln(w, string(resultJSON))
	}
	resultJSON, _ := json.Marshal(newImage)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(resultJSON))
}

func requestImgID(w http.ResponseWriter, r *http.Request) {
	response := JSONlist{}
	collection := imageCollection{}
	vars := mux.Vars(r)
	user := vars["user"]
	name := vars["name"]
	dbname := user + ".db"
	dbkey := []byte(name)
	db, err := bolt.Open(dbname, 0644, nil)
	if err != nil {
		log.Fatal(err)
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
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintln(w, string(resultJSON))
	}
	for i := range collection.Collection {
		response.Item = append(response.Item, collection.Collection[i].URN)
	}
	resultJSON, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintln(w, string(resultJSON))
}

func newCITECollection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	name := vars["name"]
	newCITECollectionDB(user, name)
	io.WriteString(w, "success")
}

func addCITE(w http.ResponseWriter, r *http.Request) {
	// /thomas/addtoCITE?name="test"&urn="test"&internal="false"&protocol="static&location="https://digi.vatlib.it/iiifimage/MSS_Barb.lat.4/Barb.lat.4_0015.jp2/full/full/0/native.jpg"
	vars := mux.Vars(r)
	user := vars["user"]
	name := r.URL.Query().Get("name")
	name = strings.Replace(name, "\"", "", -1)
	imageurn := r.URL.Query().Get("urn")
	imageurn = strings.Replace(imageurn, "\"", "", -1)
	location := r.URL.Query().Get("location")
	location = strings.Replace(location, "\"", "", -1)
	// fmt.Println(location)
	protocol := r.URL.Query().Get("protocol")
	protocol = strings.Replace(protocol, "\"", "", -1)
	externalstr := r.URL.Query().Get("external")
	externalstr = strings.Replace(externalstr, "\"", "", -1)
	external := false
	if externalstr == "true" {
		external = true
	}
	newimage := image{URN: imageurn, External: external, Protocol: protocol, Location: location}
	// fmt.Println(user, name, newimage)
	addtoCITECollection(user, name, newimage)
	io.WriteString(w, "success")
}

func newCollection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	name := vars["name"]
	imageIDs := strings.Split(vars["urns"], ",")
	var collection imageCollection
	switch len(imageIDs) {
	case 0:
		io.WriteString(w, "failed")
		return
	case 1:
		urn := gocite.SplitCITE(imageIDs[0])
		switch {
		case urn.InValid:
			io.WriteString(w, "failed")
			return
		case urn.Object == "*":
			links, err := extractLinks(urn)
			if err != nil {
				io.WriteString(w, "failed")
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
	io.WriteString(w, "success")
}

func newWork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	if r.Method == "GET" {
		varmap := map[string]interface{}{
			"user": user,
			"port": serverIP,
		}
		t, _ := template.ParseFiles("tmpl/newWork.html")
		t.Execute(w, varmap)
	} else {
		r.ParseForm()
		// logic part of log in
		workurn := r.Form["workurn"][0]
		scheme := r.Form["scheme"][0]
		group := r.Form["workgroup"][0]
		title := r.Form["title"][0]
		version := r.Form["version"][0]
		exemplar := r.Form["exemplar"][0]
		online := r.Form["online"][0]
		language := r.Form["language"][0]
		newWork := cexMeta{URN: workurn, CitationScheme: scheme, GroupName: group, WorkTitle: title, VersionLabel: version, ExemplarLabel: exemplar, Online: online, Language: language}
		fmt.Println(newWork)
		err := newWorktoDB(user, newWork)
		if err != nil {
			io.WriteString(w, "failed")
		} else {
			io.WriteString(w, "Success")
		}
	}
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

func fieldNWA(alntext []string) [][]string {
	letters := [][]string{}
	for i := range alntext {
		charSl := strings.Split(alntext[i], "")
		letters = append(letters, charSl)
	}
	length := len(letters)
	fields := make([][]string, length)
	tmp := make([]string, length)
	for i := range letters[0] {
		allspace := true
		for j := range letters {
			tmp[j] = tmp[j] + letters[j][i]
			if letters[j][i] != " " {
				allspace = false
			}
		}
		if allspace {
			for j := range letters {
				fields[j] = append(fields[j], tmp[j])
				tmp[j] = ""
			}
		}
	}
	for j := range letters {
		fields[j] = append(fields[j], tmp[j])
	}
	return fields
}

// func addSansHyphens(s string) string {
// 	hyphen := []rune(`&shy;`)
// 	after := []rune{rune('a'), rune('ā'), rune('i'), rune('ī'), rune('u'), rune('ū'), rune('ṛ'), rune('ṝ'), rune('ḷ'), rune('ḹ'), rune('e'), rune('o'), rune('ṃ'), rune('ḥ')}
// 	notBefore := []rune{rune('ṃ'), rune('ḥ'), rune(' ')}
// 	runeSl := []rune(s)
// 	newSl := []rune{}
// 	if len(runeSl) <= 2 {
// 		return s
// 	}
// 	newSl = append(newSl, runeSl[0:2]...)

// 	for i := 2; i < len(runeSl)-2; i++ {
// 		next := false
// 		possible := false
// 		for j := range after {
// 			if after[j] == runeSl[i] {
// 				possible = true
// 			}
// 		}
// 		if !possible {
// 			newSl = append(newSl, runeSl[i])
// 			continue
// 		}
// 		for j := range notBefore {
// 			if notBefore[j] == runeSl[i+1] {
// 				next = true
// 			}
// 		}
// 		if next {
// 			newSl = append(newSl, runeSl[i])
// 			next = false
// 			continue
// 		}
// 		if runeSl[i] == rune('a') {
// 			if runeSl[i+1] == rune('i') || runeSl[i+1] == rune('u') {
// 				newSl = append(newSl, runeSl[i])
// 				continue
// 			}
// 		}
// 		if runeSl[i-1] == rune(' ') {
// 			newSl = append(newSl, runeSl[i])
// 			continue
// 		}
// 		newSl = append(newSl, runeSl[i])
// 		for k := range hyphen {
// 			newSl = append(newSl, hyphen[k])
// 		}
// 	}
// 	newSl = append(newSl, runeSl[len(runeSl)-1:]...)
// 	return string(newSl)
// }

func findSpace(runeSl []rune) (spBefore, spAfter int, newSl []rune) {
	spAfter = 0
	spBefore = 0
	for i := 0; i < len(runeSl); i++ {
		if runeSl[i] == rune(' ') {
			spBefore++
		} else {
			break
		}
	}
	for i := len(runeSl) - 1; i >= 0; i-- {
		if runeSl[i] == rune(' ') {
			spAfter++
		} else {
			break
		}
	}
	return spBefore, spAfter, runeSl[spBefore : len(runeSl)-spAfter]
}

func addSansHyphens(s string) string {
	hyphen := []rune(`&shy;`)
	after := []rune{rune('a'), rune('ā'), rune('i'), rune('ī'), rune('u'), rune('ū'), rune('ṛ'), rune('ṝ'), rune('ḷ'), rune('ḹ'), rune('e'), rune('o'), rune('ṃ'), rune('ḥ')}
	notBefore := []rune{rune('ṃ'), rune('ḥ'), rune(' ')}
	runeSl := []rune(s)
	spBefore, spAfter, runeSl := findSpace(runeSl)
	newSl := []rune{}
	if len(runeSl) <= 2 {
		return s
	}
	newSl = append(newSl, runeSl[0:2]...)

	for i := 2; i < len(runeSl)-2; i++ {
		next := false
		possible := false
		for j := range after {
			if after[j] == runeSl[i] {
				possible = true
			}
		}
		if !possible {
			newSl = append(newSl, runeSl[i])
			continue
		}
		for j := range notBefore {
			if notBefore[j] == runeSl[i+1] {
				next = true
			}
		}
		if next {
			newSl = append(newSl, runeSl[i])
			next = false
			continue
		}
		if runeSl[i] == rune('a') {
			if runeSl[i+1] == rune('i') || runeSl[i+1] == rune('u') {
				newSl = append(newSl, runeSl[i])
				continue
			}
		}
		if runeSl[i-1] == rune(' ') {
			newSl = append(newSl, runeSl[i])
			continue
		}
		newSl = append(newSl, runeSl[i])
		for k := range hyphen {
			newSl = append(newSl, hyphen[k])
		}
	}
	SpBefore := []rune{}
	SpAfter := []rune{}
	for i := 0; i < spBefore; i++ {
		SpBefore = append(SpBefore, rune(' '))
	}
	for i := 0; i < spAfter; i++ {
		SpAfter = append(SpAfter, rune(' '))
	}
	if len(runeSl) < 4 {
		newSl = append(newSl, runeSl[len(runeSl)-1:]...)
	} else {
		newSl = append(newSl, runeSl[len(runeSl)-2:]...)
	}
	newSl = append(newSl, SpAfter...)
	newSl = append(SpBefore, newSl...)
	return string(newSl)
}

func nwa(text, text2 string) []string {
	hashreg := regexp.MustCompile(`#+`)
	punctreg := regexp.MustCompile(`[^\p{L}\s#]+`)
	swirlreg := regexp.MustCompile(`{[^}]*}`)
	text = swirlreg.ReplaceAllString(text, "")
	text2 = swirlreg.ReplaceAllString(text2, "")
	start := `<div class="tile is-child" lnum="L1">`
	start2 := `<div class="tile is-child" lnum="L2">`
	end := `</div>`
	collection := []string{text, text2}
	for i := range collection {
		collection[i] = strings.ToLower(collection[i])
	}
	var basetext []Word
	var comparetext []Word
	var highlight float32

	runealn1, runealn2, _ := gonwr.Align([]rune(collection[0]), []rune(collection[1]), rune('#'), 1, -1, -1)
	aln1 := string(runealn1)
	aln2 := string(runealn2)
	aligncol := fieldNWA([]string{aln1, aln2})
	aligned1, aligned2 := aligncol[0], aligncol[1]
	for i := range aligned1 {
		tmpA := hashreg.ReplaceAllString(aligned1[i], "")
		tmpB := hashreg.ReplaceAllString(aligned2[i], "")
		tmp2A := punctreg.ReplaceAllString(tmpA, "")
		tmp2B := punctreg.ReplaceAllString(tmpB, "")
		_, _, score := gonwr.Align([]rune(tmp2A), []rune(tmp2B), rune('#'), 1, -1, -1)
		base := len([]rune(tmpA))
		if len([]rune(tmpB)) > base {
			base = len([]rune(tmpB))
		}
		switch {
		case score <= 0:
			highlight = 1.0
		case score >= base:
			highlight = 0.0
		default:
			highlight = 1.0 - float32(score)/float32(base)
		}
		basetext = append(basetext, Word{Appearance: tmpA, Id: i + 1, Alignment: i + 1, Highlight: highlight})
		comparetext = append(comparetext, Word{Appearance: tmpB, Id: i + 1, Alignment: i + 1, Highlight: highlight})

	}
	text2 = start2
	for i := range comparetext {
		s := fmt.Sprintf("%.2f", comparetext[i].Highlight)
		switch comparetext[i].Id {
		case 0:
			text2 = text2 + "<span hyphens=\"manual\" style=\"background: rgba(255, 221, 87, " + s + ");\" id=\"" + strconv.Itoa(i+1) + "\" alignment=\"" + strconv.Itoa(comparetext[i].Alignment) + "\">" + addSansHyphens(comparetext[i].Appearance) + "</span>" + " "
		default:
			text2 = text2 + "<span hyphens=\"manual\" style=\"background: rgba(255, 221, 87, " + s + ");\" id=\"" + strconv.Itoa(i+1) + "\" alignment=\"" + strconv.Itoa(comparetext[i].Alignment) + "\">" + addSansHyphens(comparetext[i].Appearance) + "</span>" + " "
		}
	}
	text2 = text2 + end

	text = start
	for i := range basetext {
		s := fmt.Sprintf("%.2f", basetext[i].Highlight)
		for j := range comparetext {
			if comparetext[j].Alignment == basetext[i].Id {
				basetext[i].Alignment = comparetext[j].Id
			}
		}
		text = text + "<span hyphens=\"manual\" style=\"background: rgba(255, 221, 87, " + s + ");\" + id=\"" + strconv.Itoa(basetext[i].Id) + "\" alignment=\"" + strconv.Itoa(basetext[i].Alignment) + "\">" + addSansHyphens(basetext[i].Appearance) + "</span>" + " "
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

// multi alignment testing

type Alignments struct {
	Alignment []Alignment
	Name      []string
}

type Alignment struct {
	Source []string
	Target []string
	Score  []float32
}

func MultiPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	urn := vars["urn"]

	dbname := user + ".db"

	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"
	work := strings.Join(strings.Split(strings.Split(requestedbucket, ":")[3], ".")[0:1], ".")
	retrieveddata := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedjson := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)
	id1 := retrievedjson.URN
	text1 := retrievedjson.Text
	next1 := retrievedjson.Next
	first1 := retrievedjson.First
	last1 := retrievedjson.Last
	previous1 := retrievedjson.Previous
	swirlreg := regexp.MustCompile(`{[^}]*}`)
	text1 = swirlreg.ReplaceAllString(text1, "")
	text1 = strings.Replace(text1, "-NEWLINE-", "", -1)
	ids := []string{}
	texts := []string{}
	passageId := strings.Split(urn, ":")[4]

	buckets := Buckets(dbname)
	db, err := bolt.Open(dbname, 0644, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	for i := range buckets {
		if buckets[i] == requestedbucket {
			continue
		}
		if !gocite.IsCTSURN(buckets[i]) {
			continue
		}
		if strings.Join(strings.Split(strings.Split(buckets[i], ":")[3], ".")[0:1], ".") != work {
			continue
		}
		db.View(func(tx *bolt.Tx) error {
			// Assume bucket exists and has keys
			b := tx.Bucket([]byte(buckets[i]))

			c := b.Cursor()

			for k, v := c.First(); k != nil; k, v = c.Next() {
				retrievedjson := BoltURN{}
				json.Unmarshal([]byte(v), &retrievedjson)
				ctsurn := retrievedjson.URN
				text := strings.Replace(retrievedjson.Text, "-NEWLINE-", "", -1)
				if passageId != strings.Split(ctsurn, ":")[4] {
					continue
				}
				// make sure only witness that contain text are included
				if len(strings.Replace(text, " ", "", -1)) > 5 {
					ids = append(ids, ctsurn)
					texts = append(texts, text)
				}
			}

			return nil
		})
	}

	alignments := nwa2(text1, id1, texts, ids)
	slsl := [][]string{}
	for i := range alignments.Alignment {
		slsl = append(slsl, alignments.Alignment[i].Source)
	}
	reordered, ok := testStringSl(slsl)
	if !ok {
		panic(ok)
	}
	for i := range alignments.Alignment {
		newset := reordered[i]
		newsource := []string{}
		newtarget := []string{}
		newscore := []float32{}
		for j := range newset {
			tmpstr := ""
			tmpstr2 := ""
			for _, v := range newset[j] {
				tmpstr = tmpstr + alignments.Alignment[i].Source[v]
				tmpstr2 = tmpstr2 + alignments.Alignment[i].Target[v]
			}
			newsource = append(newsource, tmpstr)
			newtarget = append(newtarget, tmpstr2)
			var highlight float32
			_, _, score := gonwr.Align([]rune(tmpstr), []rune(tmpstr2), rune('#'), 1, -1, -1)
			base := len([]rune(tmpstr))
			if len([]rune(tmpstr2)) > base {
				base = len([]rune(tmpstr2))
			}
			switch {
			case score <= 0:
				highlight = 1.0
			case score >= base:
				highlight = 0.0
			default:
				highlight = 1.0 - float32(score)/float32(base)
			}
			newscore = append(newscore, highlight)
		}
		alignments.Alignment[i].Score = newscore
		alignments.Alignment[i].Source = newsource
		alignments.Alignment[i].Target = newtarget
	}
	start := `<div class="tile is-child" lnum="L`
	start1 := `<div id="`
	start2 := `" class="tile is-child" lnum="L`
	end := `</div>`
	tmpsl := []string{}
	tmpstr := start + strconv.Itoa(1) + `">`
	tmpstr2 := `<div class="items2">`

	for j, v := range alignments.Alignment[0].Source {
		var sc float32
		tmpstr2 = tmpstr2 + `<div id="crit` + strconv.Itoa(j+1) + `" class="box" style="display:none;">`
		appcrit := make(map[string]string)
		for k := range alignments.Alignment {
			sc = sc + alignments.Alignment[k].Score[j]
			if alignments.Alignment[k].Score[j] > float32(0) {
				newid := strings.Split(ids[k], ":")[3]
				newid = strings.Split(newid, ".")[2]
				item := alignments.Alignment[k].Target[j]
				newvalue := appcrit[item]
				if newvalue == "" {
					newvalue = newvalue + newid
				} else {
					newvalue = newvalue + "," + newid
				}
				appcrit[item] = newvalue
			}
		}
		for key, value := range appcrit {
			valueSl := strings.Split(value, ",")
			for _, valui := range valueSl {
				tmpstr2 = tmpstr2 + `<a href="#` + valui + `" onclick="highlfunc(this);">` + valui + `</a> `
			}
			tmpstr2 = tmpstr2 + addSansHyphens(key)
		}
		tmpstr2 = tmpstr2 + end
		sc = sc / float32(len(alignments.Alignment))
		s := fmt.Sprintf("%.2f", sc)
		tmpstr = tmpstr + "<span hyphens=\"manual\" style=\"background: rgba(255, 221, 87, " + s + ");\" id=\"" + strconv.Itoa(j+1) + "\" alignment=\"" + strconv.Itoa(j+1) + "\">" + addSansHyphens(v) + "</span>" + " "
	}
	tmpstr2 = tmpstr2 + end
	tmpstr = tmpstr + end
	tmpsl = append(tmpsl, tmpstr)
	for i := range alignments.Alignment {
		newid := strings.Split(ids[i], ":")[3]
		newid = strings.Split(newid, ".")[2]
		tmpstr := start1 + newid + start2 + strconv.Itoa(i+2) + `">`
		for j, v := range alignments.Alignment[i].Target {
			s := fmt.Sprintf("%.2f", alignments.Alignment[i].Score[j])
			tmpstr = tmpstr + "<span hyphens=\"manual\" style=\"background: rgba(165, 204, 107, " + s + ");\" id=\"" + strconv.Itoa(j+1) + "\" alignment=\"" + strconv.Itoa(j+1) + "\">" + addSansHyphens(v) + "</span>" + " "
		}
		tmpstr = tmpstr + end
		tmpsl = append(tmpsl, tmpstr)
	}

	tmpstr = `<div class="tile is-ancestor"><div class="tile is-parent column is-6"><div class="container"><div class="card is-fullwidth"><header class="card-header"><p class="card-header-title">`
	tmpstr = tmpstr + id1
	tmpstr = tmpstr + `</p></header><div class="card-content"><div class="content">`
	tmpstr = tmpstr + tmpsl[0]
	tmpstr = tmpstr + end
	tmpstr = tmpstr + end
	tmpstr = tmpstr + end
	tmpstr = tmpstr + end
	tmpstr = tmpstr + end
	tmpstr = tmpstr + `<div class="tile is-parent column is-6"><div class="container"><div id="trmenu">`
	for _, v := range ids {
		newid := strings.Split(v, ":")[3]
		newid = strings.Split(newid, ".")[2]
		// if i == 0 {
		// 	tmpstr = tmpstr + `<a class="button is-primary" href="#` + newid + `">` + newid + `</a>`
		// 	continue
		// }
		tmpstr = tmpstr + `<a class="button" id="button_` + newid + `" href="#` + newid + `" onclick="highlfunc(this);">` + newid + `</a>`
	}
	tmpstr = tmpstr + end
	tmpstr = tmpstr + `<div class="items">`
	for i, v := range tmpsl {
		if i == 0 {
			continue
		}
		tmpstr = tmpstr + v
	}
	tmpstr = tmpstr + end
	tmpstr = tmpstr + end
	tmpstr = tmpstr + end
	tmpstr = tmpstr + end

	tmpstr = tmpstr + `<div class="tile is-ancestor"><div class="tile is-parent"><div class="container">` + tmpstr2 + end + end + end

	transcription := Transcription{
		CTSURN:        urn,
		Transcriber:   user,
		Next:          next1,
		Previous:      previous1,
		First:         first1,
		Last:          last1,
		Transcription: tmpstr}
	port := ":7000"
	p, _ := loadMultiPage(transcription, port)
	renderTemplate(w, "multicompare", p)
}

func loadMultiPage(transcription Transcription, port string) (*Page, error) {
	user := transcription.Transcriber
	return &Page{User: user, Title: transcription.CTSURN, TextHTML: template.HTML(transcription.Transcription), Next: transcription.Next, Previous: transcription.Previous, First: transcription.First, Last: transcription.Last, Port: port}, nil
}

func fieldNWA2(alntext []string) [][]string {
	letters := [][]string{}
	for i := range alntext {
		charSl := strings.Split(alntext[i], "")
		letters = append(letters, charSl)
	}
	length := len(letters)
	fields := make([][]string, length)
	tmp := make([]string, length)
	for i := range letters[0] {
		allspace := true
		for j := range letters {
			tmp[j] = tmp[j] + letters[j][i]
			if letters[j][i] != " " {
				allspace = false
			}
		}
		if allspace {
			for j := range letters {
				fields[j] = append(fields[j], tmp[j])
				tmp[j] = ""
			}
		}
	}
	for j := range letters {
		fields[j] = append(fields[j], tmp[j])
	}
	for i := range fields {
		fields[i][0] = strings.TrimLeft(fields[i][0], " ")
	}
	return fields
}

func nwa2(basetext, baseid string, texts, ids []string) (alignments Alignments) {
	hashreg := regexp.MustCompile(`#+`)
	punctreg := regexp.MustCompile(`[^\p{L}\s#]+`)
	swirlreg := regexp.MustCompile(`{[^}]*}`)
	var highlight float32

	for i := range texts {
		alignment := Alignment{}
		texts[i] = strings.ToLower(texts[i])
		texts[i] = strings.TrimSpace(texts[i])
		texts[i] = swirlreg.ReplaceAllString(texts[i], "")
		runealn1, runealn2, _ := gonwr.Align([]rune(basetext), []rune(texts[i]), rune('#'), 1, -1, -1)
		aln1 := string(runealn1)
		aln2 := string(runealn2)
		aligncol := fieldNWA2([]string{aln1, aln2})
		aligned1, aligned2 := aligncol[0], aligncol[1]
		for j := range aligned1 {
			tmpA := hashreg.ReplaceAllString(aligned1[j], "")
			tmpB := hashreg.ReplaceAllString(aligned2[j], "")
			tmp2A := punctreg.ReplaceAllString(tmpA, "")
			tmp2B := punctreg.ReplaceAllString(tmpB, "")
			_, _, score := gonwr.Align([]rune(tmp2A), []rune(tmp2B), rune('#'), 1, -1, -1)
			base := len([]rune(tmpA))
			if len([]rune(tmpB)) > base {
				base = len([]rune(tmpB))
			}
			switch {
			case score <= 0:
				highlight = 1.0
			case score >= base:
				highlight = 0.0
			default:
				highlight = 1.0 - float32(score)/float32(base)
			}
			alignment.Source = append(alignment.Source, tmpA)
			alignment.Target = append(alignment.Target, tmpB)
			alignment.Score = append(alignment.Score, highlight)
		}
		newID := baseid + "+" + ids[i]
		alignments.Name = append(alignments.Name, newID)
		alignments.Alignment = append(alignments.Alignment, alignment)
	}
	return alignments
}

func testString(str string, strsl1 []string, cursorIn int) (cursorOut int, sl []int, ok bool) {
	calcStr1 := ""
	if len([]rune(str)) > len([]rune(strings.Join(strsl1[cursorIn:], ""))) {
		return 0, []int{}, false
	}
	base := cursorIn
	for i, v := range strsl1[cursorIn:] {
		calcStr1 = calcStr1 + v
		if calcStr1 != str {
			if i+1 == len(sl) {
				return 0, []int{}, false
			}
			sl = append(sl, i+base)
			continue
		}
		if calcStr1 == str {
			sl = append(sl, i+base)
			cursorOut = i + base + 1
			ok = true
			return cursorOut, sl, ok
		}
	}
	return 0, []int{}, false
}

func testAllTheSame(testset [][]string) bool {
	teststr := strings.Join(testset[0], "")
	for i := range testset {
		if i == 0 {
			continue
		}
		if teststr != strings.Join(testset[i], "") {
			return false
		}
	}
	return true
}

func testStringSl(slsl [][]string) (slsl2 [][][]int, ok bool) {
	if len(slsl) == 0 {
		// fmt.Println("zero length")
		slsl2 = [][][]int{}
		return slsl2, ok
	}
	ok = testAllTheSame(slsl)
	if !ok {
		// fmt.Println("slices not same length")
		slsl2 = [][][]int{}
		return slsl2, ok
	}
	// fmt.Println("passed testAllTheSame")

	length := len(slsl)

	base := make([]int, length)
	cursor := make([]int, length)
	indeces := make([][]int, length)
	testr := ""
	slsl2 = make([][][]int, length)

	for i := 0; i < len(slsl[0]); i++ {
		match := false
		indeces[0] = append(indeces[0], i)
		testr = testr + slsl[0][i]
		// fmt.Println("test", testr)
		// fmt.Scanln()

		for k := range slsl {
			if k == 0 {
				continue
			}
			cursor[k], indeces[k], match = testString(testr, slsl[k], base[k])
			if !match {
				// fmt.Println(testr, "and", slsl[k][base[k]:], "do not match")
				// fmt.Scanln()
				break
			}
		}
		if match {
			// fmt.Println("write to slice!!")
			// fmt.Scanln()
			for k := range slsl {
				slsl2[k] = append(slsl2[k], indeces[k])
				if k == 0 {
					continue
				}
				base[k] = cursor[k]
			}
			indeces[0] = []int{}
			testr = ""
		}
	}
	ok = true
	return slsl2, ok
}

// old stuff

// 	calcStr1 := ""
// 	calcStr2 := ""
// 	tmpstr := ""
// 	accessed := false
// 	count := 0

// 	length := len(slsl)

// 	base := make([]int, length)
// 	cursor := make([]int, length)
// 	indeces := make([][]int, length)
// 	slsl2 = make([][][]int, length)

// 	for i, v := range slsl[0][base[0]:] {
// 		match := false
// 		smaller := false
// 		calcStr1 = calcStr1 + v
// 		if len([]rune(calcStr1)) < len([]rune(calcStr2)) {
// 			cursor[0]++
// 			indeces[0] = append(indeces[0], i)
// 			continue
// 		}
// 		for j, w := range slsl[1][base[1]:] {
// 			tmpstr = calcStr2
// 			calcStr2 = calcStr2 + w
// 			fmt.Println("compare", calcStr1, "and", calcStr2)
// 			fmt.Scanln()
// 			if len([]rune(calcStr1)) < len([]rune(calcStr2)) {
// 				smaller = true
// 				if accessed {
// 					calcStr2 = tmpstr
// 					// accessed = false
// 				} else {
// 					calcStr2 = ""
// 				}

// 				break
// 			}
// 			if len([]rune(calcStr1)) > len([]rune(calcStr2)) {
// 				fmt.Println(len([]rune(calcStr1)), "and", len([]rune(calcStr2)))
// 				fmt.Scanln()
// 				cursor[1]++
// 				count++
// 				indeces[1] = append(indeces[1], j+base[1])
// 				continue
// 			}
// 			if calcStr1 == calcStr2 {
// 				fmt.Println("strings 1 + 2 match")
// 				fmt.Scanln()
// 				for k := range slsl {
// 					if k < 2 {
// 						continue
// 					}
// 					cursor[k], indeces[k], match = testString(calcStr1, slsl[k], base[k])
// 					if !match {
// 						fmt.Println(calcStr1, "and", slsl[k][base[k]:], "do not match")
// 						fmt.Scanln()
// 						accessed = true
// 						break
// 					}

// 				}

// 				indeces[0] = append(indeces[0], i)
// 				indeces[1] = append(indeces[1], j+base[1])
// 				cursor[1]++
// 				cursor[0]++
// 				count = 0
// 				base[1] = cursor[1]
// 				base[0] = cursor[0]
// 				break
// 			}
// 			break
// 		}
// 		if smaller {
// 			fmt.Println("smaller. count:", count)
// 			cursor[0]++
// 			cursor[1] = cursor[1] - count
// 			base[1] = cursor[1]
// 			fmt.Println("restart with", slsl[1][base[1]])
// 			fmt.Scanln()
// 			count = 0
// 			indeces[1] = []int{}
// 			indeces[0] = append(indeces[0], i)
// 			continue
// 		}
// 		if match {
// 			fmt.Println("write to slice!!")
// 			fmt.Scanln()
// 			accessed = false
// 			count = 0
// 			for k := range slsl {
// 				slsl2[k] = append(slsl2[k], indeces[k])
// 				if k < 2 {
// 					continue
// 				}
// 				base[k] = cursor[k]
// 			}

// 			indeces[0] = []int{}
// 			indeces[1] = []int{}
// 			calcStr1 = ""
// 			calcStr2 = ""

// 			if base[0] == len(slsl[0]) {
// 				ok = true
// 				return slsl2, ok
// 			}
// 			continue
// 		}

// 	}
// 	fmt.Println("!!! accessed this!!")
// 	fmt.Scanln()
// 	ok = false
// 	for k := range slsl {
// 		slsl2[k] = [][]int{}
// 	}
// 	return slsl2, ok
// }
