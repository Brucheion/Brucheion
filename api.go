package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
)

// handleCEXUpload reads a CEX file transferred in a POST request into memory
// and attempts to load the contained CEX data into the user database.
// adapted example from <https://tutorialedge.net/golang/go-file-upload-tutorial/>
func handleCEXUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	user, err := getSessionUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", 401)
		return
	}

	// max. 20mb in size
	r.ParseMultipartForm(20 << 20)
	file, handler, err := r.FormFile("file")
	if err != nil {
		respondWithError(w, "file_not_found", 400)
		return
	} else if filepath.Ext(handler.Filename) != ".cex" {
		respondWithError(w, "bad_file_ext", 400)
		return
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Printf("Error reading file:\n%s\n", err.Error())
		respondWithError(w, "bad_file_body", 500)
		return
	}

	// loadCEX currently does not particular error cases and thus might panic
	// on malformed file input. We'll handle any panics here in order to
	// provide proper responses.
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			log.Printf("Error loading file:\n%s\n", err.Error())
			respondWithError(w, "bad_cex_data", 500)
		}
	}()
	err = loadCEX(string(data), user)
	if err != nil {
		log.Printf("Error loading file:\n%s\n", err.Error())
		respondWithError(w, "bad_cex_data", 500)
		return
	}

	respondWithSuccess(w)
}

type JSONResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func respondWithSuccess(w http.ResponseWriter) {
	respondWithJSON(w, "success", "", nil, 200)
}

func respondWithError(w http.ResponseWriter, message string, code int) {
	respondWithJSON(w, "error", message, nil, code)
}

func respondWithData(w http.ResponseWriter, data interface{}, code int) {
	respondWithJSON(w, "success", "", data, code)
}

// respondWithJSON responds to a request with a JSON-encoded response of JSONResponse.
// adapted from <https://stackoverflow.com/a/59764037>
func respondWithJSON(w http.ResponseWriter, status string, message string, data interface{}, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)

	json.NewEncoder(w).Encode(JSONResponse{
		Status:  status,
		Message: message,
		Data:    data,
	})
}

// requireAuth is a middleware for wrapping an http.HandlerFunc
// and checking for a valid user session. The username is then
// stored in the request context.
func requireAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := getSession(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, _, loggedIn := testLoginStatus("", session)
		if !loggedIn {
			Logout(w, r)
			return
		}

		c := context.WithValue(r.Context(), "session", session)
		r = r.WithContext(c)
		h(w, r)
	}
}
