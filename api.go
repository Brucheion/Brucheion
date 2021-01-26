package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

// somewhat naive & pessimistic
var isSafeFileName = regexp.MustCompile(`^[a-zA-Z0-9-_\.]+$`).MatchString

type JSONExistsResponse struct {
	Exists bool `json:"exists"`
}

func handleCEXExists(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		respondWithError(w, "Parameter `name` is missing", 400)
		return
	} else if !isSafeFileName(name) {
		respondWithError(w, "Parameter `name` contains invalid characters", 400)
		return
	}

	p := filepath.Join(dataPath, "cex", name+".cex")
	_, err := os.Stat(p)
	exists := !os.IsNotExist(err)

	respondWithData(w, JSONExistsResponse{Exists: exists}, 200)
}

type JSONResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func respondWithError(w http.ResponseWriter, message string, code int) {
	respondWithJSON(w, "error", message, nil, code)
}

func respondWithData(w http.ResponseWriter, data interface{}, code int) {
	respondWithJSON(w, "success", "", data, code)
}

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
