package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"

	"github.com/ThomasK81/gocite"
	"github.com/ThomasK81/gonwr"

	"github.com/gorilla/mux"
)

type numberedLine struct {
	number int
	text   string
}

func makeNumberedLines(passage []string) []numberedLine {
	passagePattern := regexp.MustCompile("{[a-zA-Z0-9]+[_\\d+]*?[rv,]}")
	i := 1
	p := make([]numberedLine, 0, len(passage))
	for _, line := range passage {
		if passagePattern.MatchString(line) {
			i = 1
		}
		p = append(p, numberedLine{number: i, text: line})
		i += 1
	}
	return p
}

// ViewPage prepares, loads, and renders the Passage Overview
//todo: overhaul with new database functions
func ViewPage(res http.ResponseWriter, req *http.Request) {
	//First get the session..
	log.Println("Im in viewpage!!!")
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("ViewPage", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	urn := vars["urn"]
	dbname := user + ".db"

	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata, _ := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedcat, _ := BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson := BoltCatalog{}
	retrievedPassage := gocite.Passage{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedPassage)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	ctsurn := retrievedPassage.PassageID
	text := retrievedPassage.Text.TXT
	passages := strings.Split(text, "\r\n")
	text = ""
	/* for i, v := range passages {
	} */
	// line_nums_by_folio := countPassageLineNums(passages)
	numberedLines := makeNumberedLines(passages)
	for _, line := range numberedLines {
		text += `
			<p name="textpassage" style="padding: 0 0 0.25em 0">
				<span style="font-weight: bold">` + strconv.Itoa(line.number) + `: </span>
				` + line.text + `
			</p>`
	}
	previous := retrievedPassage.Prev.PassageID
	next := retrievedPassage.Next.PassageID
	imageref := []string{}
	for _, tmp := range retrievedPassage.ImageLinks {
		imageref = append(imageref, tmp.Object)
	}
	/*first := retrievedPassage.First.PassageID
	last := retrievedPassage.Last.PassageID*/
	work, _ := BoltRetrieveWork(dbname, requestedbucket)
	first := work.First.PassageID
	last := work.Last.PassageID
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

	kind := "/view/"
	page, _ := loadPage(transcription, kind)
	renderTemplate(res, "view", page)
}

//todo: needs overhaul with new database functions
func comparePage(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("comparePage", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	urn := vars["urn"]
	urn2 := vars["urn2"]
	dbname := user + ".db"

	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata, _ := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedcat, _ := BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson := BoltCatalog{}
	//retrievedPassage := BoltURN{}
	retrievedPassage := gocite.Passage{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedPassage)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	ctsurn := retrievedPassage.PassageID
	text := retrievedPassage.Text.TXT
	previous := retrievedPassage.Prev.PassageID
	next := retrievedPassage.Next.PassageID
	imageref := []string{}
	for _, tmp := range retrievedPassage.ImageLinks {
		imageref = append(imageref, tmp.Object)
	}
	/*first := retrievedPassage.First.PassageID
	last := retrievedPassage.Last.PassageID*/
	work, _ := BoltRetrieveWork(dbname, requestedbucket)
	first := work.First.PassageID
	last := work.Last.PassageID
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
	retrieveddata, _ = BoltRetrieve(dbname, requestedbucket, urn2)
	retrievedcat, _ = BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson = BoltCatalog{}
	//retrievedPassage = BoltURN{}
	retrievedPassage = gocite.Passage{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedPassage)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	ctsurn = retrievedPassage.PassageID
	text = retrievedPassage.Text.TXT
	previous = retrievedPassage.Prev.PassageID
	next = retrievedPassage.Next.PassageID
	imageref = []string{}
	for _, tmp := range retrievedPassage.ImageLinks {
		imageref = append(imageref, tmp.Object)
	}
	work, _ = BoltRetrieveWork(dbname, requestedbucket)
	first = work.First.PassageID
	last = work.Last.PassageID
	/*first = retrievedPassage.First.PassageID
	last = retrievedPassage.Last.PassageID*/
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

	compPage, _ := loadCompPage(transcription, transcription2)
	renderCompTemplate(res, "compare", compPage)
}

func consolidatePage(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("consolidatePage", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	urn := vars["urn"]
	urn2 := vars["urn2"]
	dbname := user + ".db"

	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata, _ := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedcat, _ := BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson := BoltCatalog{}
	retrievedPassage := BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedPassage)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	ctsurn := retrievedPassage.URN
	text := ""
	linetext := retrievedPassage.LineText
	for i := range linetext {
		text = text + linetext[i]
		if i < len(linetext)-1 {
			text = text + " "
		}
	}
	previous := retrievedPassage.Previous
	next := retrievedPassage.Next
	imageref := retrievedPassage.ImageRef
	first := retrievedPassage.First
	last := retrievedPassage.Last
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
	retrieveddata, _ = BoltRetrieve(dbname, requestedbucket, urn2)
	retrievedcat, _ = BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedcatjson = BoltCatalog{}
	retrievedPassage = BoltURN{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedPassage)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	ctsurn = retrievedPassage.URN
	text = ""
	linetext = retrievedPassage.LineText
	for i := range linetext {
		text = text + linetext[i]
		if i < len(linetext)-1 {
			text = text + " "
		}
	}
	previous = retrievedPassage.Previous
	next = retrievedPassage.Next
	imageref = retrievedPassage.ImageRef
	first = retrievedPassage.First
	last = retrievedPassage.Last
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

	compPage, _ := loadCompPage(transcription, transcription2)
	renderCompTemplate(res, "consolidate", compPage)
}

//EditPage prepares, loads, and renders the Transcription Desk
//can possibly be overhauled using gocite release 2.0.0
func EditPage(res http.ResponseWriter, req *http.Request) {
	log.Println("Im in EditPage!!!")
	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("EditPage", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	urn := vars["urn"]
	dbname := user + ".db"
	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"
	work, _ := BoltRetrieveWork(dbname, requestedbucket)
	// adding testing if requestedbucket exists...
	retrieveddata, _ := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedPassage := gocite.Passage{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedPassage)

	text := retrievedPassage.Text.TXT
	imageref := []string{}
	for _, tmp := range retrievedPassage.ImageLinks {
		imageref = append(imageref, tmp.Object)
	}
	imagejs := "urn:cite2:test:googleart.positive:DuererHare1502"

	switch len(imageref) > 0 {
	case true:
		imagejs = imageref[0]
	}

	transcription := Transcription{
		CTSURN:        retrievedPassage.PassageID,
		Transcriber:   user,
		Transcription: text,
		Previous:      retrievedPassage.Prev.PassageID,
		Next:          retrievedPassage.Next.PassageID,
		First:         work.First.PassageID,
		Last:          work.Last.PassageID,
		TextRef:       textref,
		ImageRef:      imageref,
		ImageJS:       imagejs}

	kind := "/edit/"
	page, _ := loadPage(transcription, kind)
	renderTemplate(res, "edit", page)
}

//EditPageFormat prepares, loads, and renders the Transcription Desk with format
//can possibly be overhauled using gocite release 2.0.0
func EditPageFormat(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("EditPage", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	urn := vars["urn"]
	format := vars["format"]
	dbname := user + ".db"
	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"
	work, _ := BoltRetrieveWork(dbname, requestedbucket)
	// adding testing if requestedbucket exists...
	retrieveddata, _ := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedPassage := gocite.Passage{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedPassage)

	text := retrievedPassage.Text.TXT
	imageref := []string{}
	for _, tmp := range retrievedPassage.ImageLinks {
		imageref = append(imageref, tmp.Object)
	}
	imagejs := "urn:cite2:test:googleart.positive:DuererHare1502"

	switch len(imageref) > 0 {
	case true:
		imagejs = imageref[0]
	}

	transcription := Transcription{
		CTSURN:        retrievedPassage.PassageID,
		Transcriber:   user,
		Transcription: text,
		Previous:      retrievedPassage.Prev.PassageID,
		Next:          retrievedPassage.Next.PassageID,
		First:         work.First.PassageID,
		Last:          work.Last.PassageID,
		TextRef:       textref,
		ImageRef:      imageref,
		ImageJS:       imagejs}

	kind := "/edit/"
	page, _ := loadPage(transcription, kind)
	if format == "pt" {
		renderTemplate(res, "editpt", page)
	} else {
		renderTemplate(res, "edit", page)
	}
}

//Edit2Page prepares, loads, and renders the Image Citation Editor
func Edit2Page(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("Edit2Page", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	urn := vars["urn"]
	dbname := user + ".db"
	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata, _ := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedPassage := gocite.Passage{}
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedPassage)

	ctsurn := retrievedPassage.PassageID
	text := retrievedPassage.Text.TXT
	previous := retrievedPassage.Prev.PassageID
	next := retrievedPassage.Next.PassageID
	imageref := []string{}
	for _, tmp := range retrievedPassage.ImageLinks {
		imageref = append(imageref, tmp.Object)
	}
	/*First := retrievedPassage.First.PassageID
	last := retrievedPassage.Last.PassageID*/
	work, _ := BoltRetrieveWork(dbname, requestedbucket)
	first := work.First.PassageID
	last := work.Last.PassageID
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
	kind := "/edit2/"
	page, _ := loadPage(transcription, kind)
	renderTemplate(res, "edit2", page)
}

//EditCatPage prepares, loads, and renders the Edit Metadata page
func EditCatPage(res http.ResponseWriter, req *http.Request) {
	log.Println("Im in EditPageEditCatPage!!!")
	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("EditCatPage", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	urn := vars["urn"]
	dbname := user + ".db"
	textref := Buckets(dbname)
	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	// adding testing if requestedbucket exists...
	retrieveddata, _ := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedcat, _ := BoltRetrieve(dbname, requestedbucket, requestedbucket)
	retrievedWork, _ := BoltRetrieveWork(dbname, requestedbucket)
	retrievedcatjson := BoltCatalog{}
	retrievedPassage := gocite.Passage{}

	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedPassage)
	json.Unmarshal([]byte(retrievedcat.JSON), &retrievedcatjson)

	catid := retrievedcatjson.URN
	catcit := retrievedcatjson.Citation
	catgroup := retrievedcatjson.GroupName
	catwork := retrievedcatjson.WorkTitle
	catversion := retrievedcatjson.VersionLabel
	catexpl := retrievedcatjson.ExemplarLabel
	caton := retrievedcatjson.Online
	catlan := retrievedcatjson.Language
	transcription := Transcription{
		CTSURN:      retrievedPassage.PassageID,
		Transcriber: user,
		TextRef:     textref,
		Previous:    retrievedPassage.Prev.PassageID,
		Next:        retrievedPassage.Next.PassageID,
		First:       retrievedWork.First.PassageID,
		Last:        retrievedWork.Last.PassageID,
		CatID:       catid, CatCit: catcit, CatGroup: catgroup, CatWork: catwork, CatVers: catversion, CatExmpl: catexpl, CatOn: caton, CatLan: catlan}
	kind := "/editcat/"
	page, _ := loadPage(transcription, kind)
	renderTemplate(res, "editcat", page)
}

//MultiPage prepares, loads, and renders the Multicompare page
func MultiPage(res http.ResponseWriter, req *http.Request) {
	log.Println("Im in MultiPage!!!")

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("MultiPage", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	urn := vars["urn"]

	dbname := user + ".db"

	vquery := req.URL.Query()
	keep := vquery.Get("keep")

	var alignments Alignments

	requestedbucket := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"
	work := strings.Join(strings.Split(strings.Split(requestedbucket, ":")[3], ".")[0:1], ".")
	retrieveddata, _ := BoltRetrieve(dbname, requestedbucket, urn)
	retrievedPassage := gocite.Passage{}
	retrievedWork, _ := BoltRetrieveWork(dbname, requestedbucket)
	json.Unmarshal([]byte(retrieveddata.JSON), &retrievedPassage)
	id1 := retrievedPassage.PassageID
	text1 := retrievedPassage.Text.TXT
	if config.UseNormalization && retrievedPassage.Text.Normalised != "" {
		text1 = retrievedPassage.Text.Normalised
	} // config setting updated only on restart since loadConfiguration() in brucheion.go
	next1 := retrievedPassage.Next.PassageID
	previous1 := retrievedPassage.Prev.PassageID
	first1 := retrievedWork.First.PassageID
	last1 := retrievedWork.Last.PassageID
	ids := []string{}
	texts := []string{}
	passageID := strings.Split(urn, ":")[4]

	buckets := Buckets(dbname)
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		log.Println(fmt.Printf("MultiPage: error opening userDB: %s", err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
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
				retrievedPassage := gocite.Passage{}
				json.Unmarshal([]byte(v), &retrievedPassage)
				ctsurn := retrievedPassage.PassageID
				if ctsurn == "" {
					continue
				}
				if passageID != strings.Split(ctsurn, ":")[4] {
					continue
				}
				text := retrievedPassage.Text.TXT
				if config.UseNormalization && retrievedPassage.Text.Normalised != "" {
					text = retrievedPassage.Text.Normalised
				} // config setting updated only on restart since loadConfiguration() in brucheion.go

				// make sure only witness that contain text are included
				if len(strings.Replace(text, " ", "", -1)) > 5 {
					ids = append(ids, ctsurn)
					texts = append(texts, text)
				}
			}

			return nil
		})
	}
	db.Close()

	switch keep == "true" {
	case true:
		dbkey := []byte(id1)
		db, err := openBoltDB(dbname) //open bolt DB using helper function
		if err != nil {
			log.Println(fmt.Printf("requestImgID: error opening userDB: %s", err))
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		err = db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte("alignmentsCollection"))
			if bucket == nil {
				return errors.New("failed to get bucket")
			}
			val := bucket.Get(dbkey)
			if val == nil {
				return errors.New("failed to retrieve value")
			}
			alignments, err = gobDecodeAlignments(val)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			log.Println(fmt.Printf("error retrieving alignments: %s", err))
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		db.Close()
	default:
		alignments = nwa2(text1, id1, texts, ids)
		aligntime := time.Now()
		alignments.AlignmentTime = aligntime.Format("20060102150405")
		alignments.AlignmentID = id1
		// building the lemmata
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
					highlight = 1.0 - float32(score)/(3*float32(base))
				}
				newscore = append(newscore, highlight)
			}
			alignments.Alignment[i].Score = newscore
			alignments.Alignment[i].Source = newsource
			alignments.Alignment[i].Target = newtarget
		}

		AlignmentsToDB(dbname, alignments)
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
		tmpstr2 = tmpstr2 + `<div id="crit` + strconv.Itoa(j+1) + `" class="content" style="display:none;">`
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
		appcount := 1
		for key, value := range appcrit {
			tmpstr2 = tmpstr2 + strconv.Itoa(appcount) + "."
			valueSl := strings.Split(value, ",")
			for _, valui := range valueSl {
				tmpstr2 = tmpstr2 + `<a href="#` + valui + `" onclick="highlfunc(this);">` + valui + `</a> `
			}
			tmpstr2 = tmpstr2 + addSansHyphens(key) + `<br/>`
			appcount++
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
		tmpstr = tmpstr + `<br><br/><a class="button is-small is-primary" href = "#" onClick="MyWindow=window.open('` + config.Host + "/view/" + ids[i] + `','MyWindow'); return false;">PassageView</a>` + end
		tmpsl = append(tmpsl, tmpstr)
	}

	tmpstr = `<div class="tile is-ancestor"><div class="tile is-parent column is-6"><div class="container"><div class="card is-fullwidth"><header class="card-header"><p class="card-header-title">Text</p></header><div class="card-content"><div class="content">`
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

	tmpstr = tmpstr + `<div class="tile is-ancestor"><div class="tile is-parent column is-6"><div class="container"><div class="card"><header class="card-header"><p class="card-header-title">Variants</p></header><div class="card-content">` + tmpstr2 + end + end + end + end + end
	transcription := Transcription{
		CTSURN:        urn,
		Transcriber:   user,
		TextRef:       buckets,
		Next:          next1,
		Previous:      previous1,
		First:         first1,
		Last:          last1,
		Transcription: tmpstr}
	page, _ := loadMultiPage(transcription)
	renderTemplate(res, "multicompare", page)
}

//SeeAlignment prepares, loads, and renders the SeeAlignment page
func SeeAlignment(res http.ResponseWriter, req *http.Request) {
	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("MultiPage", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	urn := vars["urn"]
	var alignments Alignments

	dbname := user + ".db"
	dbkey := []byte(urn)
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		log.Println(fmt.Printf("requestImgID: error opening userDB: %s", err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("alignmentsCollection"))
		if bucket == nil {
			return errors.New("failed to get bucket")
		}
		val := bucket.Get(dbkey)
		if val == nil {
			return errors.New("failed to retrieve value")
		}
		alignments, err = gobDecodeAlignments(val)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Println(fmt.Printf("error retrieving alignments: %s", err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	db.Close()
	// add error handling
	jsonresponse, _ := json.Marshal(alignments)
	io.WriteString(res, string(jsonresponse))
}

//TableAlignments prepares, loads, and renders the TableAlignments page
func TableAlignments(res http.ResponseWriter, req *http.Request) {
	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("MultiPage", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	urn := vars["urn"]
	var alignments Alignments

	dbname := user + ".db"
	dbkey := []byte(urn)
	db, err := openBoltDB(dbname) //open bolt DB using helper function
	if err != nil {
		log.Println(fmt.Printf("requestImgID: error opening userDB: %s", err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("alignmentsCollection"))
		if bucket == nil {
			return errors.New("failed to get bucket")
		}
		val := bucket.Get(dbkey)
		if val == nil {
			return errors.New("failed to retrieve value")
		}
		alignments, err = gobDecodeAlignments(val)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Println(fmt.Printf("error retrieving alignments: %s", err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	db.Close()
	type TableData struct {
		TableID   template.HTML
		TableHead template.HTML
		TableBody template.HTML
		Host      string
	}
	tableid := `<a href="` + config.Host + `/view/` + alignments.AlignmentID + `" target="_blank">` + alignments.AlignmentID + `</a>`
	tablehead := `<tr><th></th>`
	for _, v := range alignments.Name {
		tablehead = tablehead + `<th><a href="` + config.Host + `/view/` + v + `" target="_blank">` + v + `</a></th>`
	}
	tablehead = tablehead + `</tr>`
	tablebody := ``
	for i := range alignments.Alignment[0].Source {
		tablebody = tablebody + `<tr>`
		tablebody = tablebody + `<td class = "th">` + alignments.Alignment[0].Source[i] + `<i class="fa fa-plus-square" onclick="addFunction(this)"></i><i class="fa fa-minus-square" onclick="removeFunction(this)"></i></th>`
		for j := range alignments.Alignment {
			tablebody = tablebody + `<td>` + alignments.Alignment[j].Target[i] + `</td>`
		}
		tablebody = tablebody + `</tr>`
	}
	aligntable := TableData{
		TableID:   template.HTML(tableid),
		TableHead: template.HTML(tablehead),
		TableBody: template.HTML(tablebody),
		Host:      config.Host,
	}
	templates.ExecuteTemplate(res, "tablealignment.html", aligntable)
}

//TreePage prepares, loads, and renders a Morpho-syntactic Treebank page
func TreePage(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("TreePage", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	dbname := user + ".db"

	textref := Buckets(dbname)

	transcription := Transcription{
		Transcriber: user,
		TextRef:     textref}
	page, _ := loadCrudPage(transcription)
	renderTemplate(res, "tree", page)
}

func CrudPage(res http.ResponseWriter, req *http.Request) {

	log.Println("Breakpoint 1: Before getting the session")
	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Breakpoint 2: Before getting the user")

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("CrudPage", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	log.Println("Breakpoint 3: Before getting the dbname")

	dbname := user + ".db"

	log.Println("Breakpoint 4: In the middle")

	textref := Buckets(dbname)

	log.Println("Breakpoint 5: Before loading the CrudPage")

	transcription := Transcription{
		Transcriber: user,
		TextRef:     textref}
	page, _ := loadCrudPage(transcription)
	log.Println("Breakpoint 6: Before rendering the template")
	renderTemplate(res, "crud", page)
	log.Println("Breakpoint 7: after rendering the template")
}
