package main

import (
	"log"
	"net/http"
)

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

func renderLoginTemplate(res http.ResponseWriter, tmpl string, p *LoginPage) {
	err := templates.ExecuteTemplate(res, tmpl+".html", p)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func renderAuthTemplate(res http.ResponseWriter, tmpl string, p *LoginPage) {
	err := templates.ExecuteTemplate(res, tmpl+".html", p)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
	log.Println("Debug: Rendering Auth Template")
}
