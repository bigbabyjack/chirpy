package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func (cfg *apiConfig) handlerCreateChirps(w http.ResponseWriter, r *http.Request) {
	// get the body of the request
	type parameters struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	// decode body
	if err != nil {
		log.Printf("Error decoding parameters %s\n", err)
		respondWithError(w, 500, "Error decoding parameters.")
	}
	// check length of chirp
	if len(params.Body) > 140 {
		log.Printf("Chirp is too long")
		respondWithError(w, 400, "Chirp is too long")
		return
	}
	// check for profanity
	profaneWords := []string{"kerfuffle", "sharbert", "fornax"}
	cleanedBody := getCleanedBody(profaneWords, params.Body)

	c, err := cfg.db.CreateChirp(cleanedBody)
	if err != nil {
		log.Printf("Error saving chirp.")
		errMsg := fmt.Sprintf("Error saving chirp: %s", err)
		respondWithError(w, 500, errMsg)
		return
	}
	respondWithJSON(w, 201, c)
	return
}
