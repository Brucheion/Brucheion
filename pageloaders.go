package main

import (
	"encoding/json"
	"html/template"
	"strconv"
	"strings"
)

func loadCrudPage(transcription Transcription) (*Page, error) {
	user := transcription.Transcriber
	var textrefrences []string
	for i := range transcription.TextRef {
		textrefrences = append(textrefrences, transcription.TextRef[i])
	}
	textref := strings.Join(textrefrences, " ")
	return &Page{User: user, Text: template.HTML(textref), Host: config.Host}, nil
}

func loadPage(transcription Transcription, kind string) (*Page, error) {
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
		previouslink = `<a href ="` + config.Host + `/new/">add previous</a>`
		previous = title
	default:
		previouslink = `<a href ="` + config.Host + kind + previous + `">` + previous + `</a>`
	}
	switch {
	case next == "":
		nextlink = `<a href ="` + config.Host + `/new/">add next</a>`
		next = title
	default:
		nextlink = `<a href ="` + config.Host + kind + next + `">` + next + `</a>`
	}
	var textrefrences []string
	for i := range transcription.TextRef {
		if transcription.TextRef[i] == "imgCollection" || transcription.TextRef[i] == "meta" {
			continue
		}
		requestedbucket := transcription.TextRef[i]
		texturn := requestedbucket + strings.Split(title, ":")[4]

		// adding testing if requestedbucket exists...
		retrieveddata := BoltRetrieve(dbname, requestedbucket, texturn)
		if retrieveddata.JSON == "" {
			continue
		}
		retrievedjson := BoltURN{}
		json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)

		ctsurn := retrievedjson.URN
		var htmllink string
		switch {
		case ctsurn == title:
			htmllink = `<option value="` + config.Host + kind + ctsurn + `" selected>` + transcription.TextRef[i] + `</option>`
		case ctsurn == "":
			ctsurn = BoltRetrieveFirstKey(dbname, requestedbucket)
			htmllink = `<option value="` + config.Host + kind + ctsurn + `">` + transcription.TextRef[i] + `</option>`
		default:
			htmllink = `<option value="` + config.Host + kind + ctsurn + `">` + transcription.TextRef[i] + `</option>`
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
		Host:         config.Host,
		ImageJS:      imagejs}, nil
}

func loadCompPage(transcription, transcription2 Transcription) (*CompPage, error) {
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
		Host:      config.Host}, nil
}

func loadMultiPage(transcription Transcription) (*Page, error) {
	user := transcription.Transcriber
	dbname := user + ".db"
	var textrefrences []string
	for i := range transcription.TextRef {
		if transcription.TextRef[i] == "imgCollection" || transcription.TextRef[i] == "meta" {
			continue
		}
		requestedbucket := transcription.TextRef[i]
		texturn := requestedbucket + strings.Split(transcription.CTSURN, ":")[4]

		// adding testing if requestedbucket exists...
		retrieveddata := BoltRetrieve(dbname, requestedbucket, texturn)
		if retrieveddata.JSON == "" {
			continue
		}
		retrievedjson := BoltURN{}
		json.Unmarshal([]byte(retrieveddata.JSON), &retrievedjson)

		ctsurn := retrievedjson.URN

		var htmllink string
		switch {
		case ctsurn == transcription.CTSURN:
			htmllink = `<option value="` + config.Host + "/multicompare/" + ctsurn + `" selected>` + transcription.TextRef[i] + `</option>`
		case ctsurn == "":
			ctsurn = BoltRetrieveFirstKey(dbname, requestedbucket)
			htmllink = `<option value="` + config.Host + "/multicompare/" + ctsurn + `">` + transcription.TextRef[i] + `</option>`
		default:
			htmllink = `<option value="` + config.Host + "/multicompare/" + ctsurn + `">` + transcription.TextRef[i] + `</option>`
		}
		textrefrences = append(textrefrences, htmllink)
	}
	textref := strings.Join(textrefrences, " ")
	texthtml := template.HTML(textref)
	return &Page{User: user, Title: transcription.CTSURN, TextHTML: texthtml, InTextHTML: template.HTML(transcription.Transcription), Next: transcription.Next, Previous: transcription.Previous, First: transcription.First, Last: transcription.Last, Host: config.Host}, nil
}
