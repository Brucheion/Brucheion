package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"text/template"
)

//newWork extracts cexMeta data from the *http.Request form values and
//passes it to newWorkToDB to save it in the user database
func newWork(res http.ResponseWriter, req *http.Request) {

	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("newWork", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
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
		err := newWorkToDB(user, newWork)
		if err != nil {
			io.WriteString(res, "failed")
		} else {
			io.WriteString(res, "Success")
		}
	}
}
