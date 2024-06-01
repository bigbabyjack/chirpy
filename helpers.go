package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type decodeErrorResponse struct {
		Error string `json:"error"`
	}
	dat, err := json.Marshal(decodeErrorResponse{Error: msg})
	if err != nil {
		respondWithInternalError(w, err)
	}
	w.WriteHeader(code)
	w.Write(dat)
	return

}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		respondWithInternalError(w, err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
	return

}

func respondWithInternalError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	log.Fatalf("Internal error: %s", err)
}
