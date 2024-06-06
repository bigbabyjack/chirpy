package main

import (
	"fmt"
	"net/http"
	"strconv"
)

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	s := r.URL.Query().Get("author_id")
	if s != "" {
		authorID, err := strconv.Atoi(s)
		if err != nil {
			respondWithError(w, 404, "User not found")
			return
		}
		chirps, err := cfg.db.GetChirpsByAuthor(authorID)
		if err != nil {
			respondWithError(w, 500, "Error getting chirps")
		}
		respondWithJSON(w, 200, chirps)
		return

	}
	chirps, err := cfg.db.GetChirps()
	if err != nil {
		respondWithError(w, 500, "Unable to retrieve chirps.")
		return
	}
	respondWithJSON(w, 200, chirps)
	return

}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	chirpID, err := strconv.Atoi(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid chirpID %v", chirpID))
		return
	}
	chirp, err := cfg.db.GetChirp(chirpID)
	if err != nil {
		respondWithError(w, 404, fmt.Sprintf("Chirp with ID %v not found", chirpID))
		return
	}
	respondWithJSON(w, 200, chirp)
	return

}
