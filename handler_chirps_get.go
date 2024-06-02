package main

import (
	"fmt"
	"net/http"
	"strconv"
)

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.GetChirps()
	if err != nil {
		respondWithError(w, 500, "Unable to retrieve chirps.")
	}
	respondWithJSON(w, 200, chirps)

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
