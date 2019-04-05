package main

import (
	"net/http"
)

func renderTemplate(res http.ResponseWriter, tmpl string, page *Page) {
	err := templates.ExecuteTemplate(res, tmpl+".html", page)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func renderCompTemplate(res http.ResponseWriter, tmpl string, compPage *CompPage) {
	err := templates.ExecuteTemplate(res, tmpl+".html", compPage)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func renderLoginTemplate(res http.ResponseWriter, tmpl string, loginPage *LoginPage) {
	err := templates.ExecuteTemplate(res, tmpl+".html", loginPage)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func renderAuthTemplate(res http.ResponseWriter, tmpl string, loginPage *LoginPage) {
	err := templates.ExecuteTemplate(res, tmpl+".html", loginPage)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
