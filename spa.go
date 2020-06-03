package main

import "net/http"

func spaHandler(res http.ResponseWriter, req *http.Request) {
	err := templates.ExecuteTemplate(res, "spa.html", nil)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
