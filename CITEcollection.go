package main

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func newCITECollection(res http.ResponseWriter, req *http.Request) {
	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("newCITECollection", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	vars := mux.Vars(req)
	name := vars["name"] //the name of the new CITE collection
	newCITECollectionToDB(user, name)
	io.WriteString(res, "success")
}

//addCITE adds a new CITE reference to the user database.
//It extracts the reference from the the http.Request and passes it to addtoCITECollection
func addCITE(res http.ResponseWriter, req *http.Request) {
	//First get the session..
	session, err := getSession(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	//..and check if user is logged in.
	user, message, loggedin := testLoginStatus("addCITE", session)
	if loggedin {
		log.Println(message)
	} else {
		log.Println(message)
		Logout(res, req)
		return
	}

	// /thomas/addtoCITE?name="test"&urn="test"&internal="false"&protocol="static"&location="https://digi.vatlib.it/iiifimage/MSS_Barb.lat.4/Barb.lat.4_0015.jp2/full/full/0/native.jpg"
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
	addImageToCITECollection(user, name, newimage)
	io.WriteString(res, "success")
}
