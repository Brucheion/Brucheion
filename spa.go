package main

import "net/http"

type staticPage struct {
	Title string
}

func createAppHandler(title string) func(http.ResponseWriter, *http.Request) {
	page := staticPage{Title: title}
	return func(res http.ResponseWriter, req *http.Request) {
		err := templates.ExecuteTemplate(res, "spa.html", page)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
	}
}
