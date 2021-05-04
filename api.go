package main

import (
	"context"
	"encoding/json"
	"github.com/ThomasK81/gocite"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

type Passage struct {
	ID                 string      `json:"id"`
	Transcriber        string      `json:"transcriber"`
	TranscriptionLines []string    `json:"transcriptionLines"`
	PreviousPassage    string      `json:"previousPassage"`
	NextPassage        string      `json:"nextPassage"`
	FirstPassage       string      `json:"firstPassage"`
	LastPassage        string      `json:"lastPassage"`
	ImageRefs          []string    `json:"imageRefs"`
	TextRefs           []string    `json:"textRefs"`
	Catalog            BoltCatalog `json:"catalog"`
}

type User struct {
	Name string `json:"name"`
}

func handleUser(w http.ResponseWriter, r *http.Request) {
	user, err := getSessionUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", 401)
		return
	}

	respondWithData(w, User{
		Name: user,
	}, 200)
}

// handlePassage retrieves a passage and associated information from the user database.
func handlePassage(w http.ResponseWriter, r *http.Request) {
	user, err := getSessionUser(r)
	if err != nil {
		http.Error(w, "Unauthorized", 401)
		return
	}

	vars := mux.Vars(r)
	urn := vars["urn"]
	if !gocite.IsCTSURN(urn) {
		http.Error(w, "Bad request", 400)
		return
	}

	dbName := user + ".db"
	textRefs := Buckets(dbName)
	bucketName := strings.Join(strings.Split(urn, ":")[0:4], ":") + ":"

	d, err := BoltRetrieve(dbName, bucketName, urn)
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}
	c, err := BoltRetrieve(dbName, bucketName, bucketName)
	if err != nil {
		http.Error(w, "Internal server error", 500)
		return
	}

	catalog := BoltCatalog{}
	passage := gocite.Passage{}
	json.Unmarshal([]byte(d.JSON), &passage)
	json.Unmarshal([]byte(c.JSON), &catalog)

	text := passage.Text.TXT
	passages := strings.Split(text, "\r\n")
	work, _ := BoltRetrieveWork(dbName, bucketName)

	var imageRefs []string
	for _, tmp := range passage.ImageLinks {
		imageRefs = append(imageRefs, tmp.Object)
	}

	p := Passage{
		ID:                 passage.PassageID,
		Transcriber:        user,
		TranscriptionLines: passages,
		PreviousPassage:    passage.Prev.PassageID,
		NextPassage:        passage.Next.PassageID,
		FirstPassage:       work.First.PassageID,
		LastPassage:        work.Last.PassageID,
		ImageRefs:          imageRefs,
		TextRefs:           textRefs,
		Catalog:            catalog,
	}

	respondWithData(w, p, 200)
}

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
