package main

import (
	"errors"
	"github.com/markbates/pkger"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

// adopted from: <https://osinet.fr/go/en/articles/bundling-templates-with-pkger/>
func compileTemplates(dir string) (*template.Template, error) {
	t := createBaseTemplate()
	err := pkger.Walk(dir, func(path string, info os.FileInfo, _ error) error {
		if info.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		name := filepath.Base(path)
		f, _ := pkger.Open(path)
		sl, _ := ioutil.ReadAll(f)

		var tmpl *template.Template
		if t == nil {
			t = template.New(name)
		}
		if name == t.Name() {
			tmpl = t
		} else {
			tmpl = t.New(name)
		}
		_, err := tmpl.Parse(string(sl))
		return err
	})
	return t, err
}
