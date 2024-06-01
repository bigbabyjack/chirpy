package main

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"strings"
)

func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
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

	respondWithJSON(w, http.StatusOK, struct {
		CleanedBody string `json:"cleaned_body"`
	}{cleanedBody})
	return

}

func getCleanedBody(profaneWords []string, body string) string {
	words := strings.Split(body, " ")
	for i, word := range words {
		if slices.Contains(profaneWords, strings.ToLower(word)) {
			words[i] = "****"
		}

	}
	cleanedBody := strings.Join(words, " ")
	return cleanedBody
}
