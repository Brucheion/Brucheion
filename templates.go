package main

import (
	"errors"
	"html/template"
	"net/http"
)

// <https://stackoverflow.com/a/18276968>
func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

func createBaseTemplate() *template.Template {
	t := template.New("")

	funcMap := template.FuncMap{}
	funcMap["dict"] = dict
	t.Funcs(funcMap)

	return t
}

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
