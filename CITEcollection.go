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
// Examples:
// localhost:7000/addtoCITE?name="urn:cite2:iiifimages:test:"&urn="urn:cite2:iiifimages:test:1"&external="true"&protocol="iiif"&location="https://libimages1.princeton.edu/loris/pudl0001%2F4609321%2Fs42%2F00000004.jp2/info.json"
// localhost:7000/addtoCITE?name="urn:cite2:staticimages:test:"&urn="urn:cite2:staticimages:test:1"&external="true"&protocol="static"&location="https://upload.wikimedia.org/wikipedia/commons/8/81/Rembrandt_The_Three_Crosses_1653.jpg"
// localhost:7000/addtoCITE?name="urn:cite2:dzi:test:"&urn="urn:cite2:nyaya:M3img.positive:m3_098"&external="false"&protocol="localDZ"&location="urn:cite2:nyaya:M3img.positive:m3_098"
// localhost:7000/addtoCITE?name="urn:cite2:mixedimages:test:"&urn="urn:cite2:iiifimages:test:1"&external="true"&protocol="iiif"&location="https://libimages1.princeton.edu/loris/pudl0001%2F4609321%2Fs42%2F00000004.jp2/info.json"
// localhost:7000/addtoCITE?name="urn:cite2:mixedimages:test:"&urn="urn:cite2:staticimages:test:1"&external="true"&protocol="static"&location="https://upload.wikimedia.org/wikipedia/commons/8/81/Rembrandt_The_Three_Crosses_1653.jpg"
// localhost:7000/addtoCITE?name="urn:cite2:mixedimages:test:"&urn="urn:cite2:nyaya:M3img.positive:m3_098"&external="false"&protocol="localDZ"&location="urn:cite2:nyaya:M3img.positive:m3_098"
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
