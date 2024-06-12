package main

import (
	"fmt"
	"net/http"
	"strconv"
)

func parseAuthorID(r *http.Request) (int, error) {
	s := r.URL.Query().Get("author_id")
	if s != "" {
		authorID, err := strconv.Atoi(s)
		if err != nil {
			return 0, err
		}
		return authorID, nil
	}
	return 0, nil
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	authorID, err := parseAuthorID(r)
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}

	if authorID != 0 {
		chirps, err := cfg.db.GetChirpsByAuthor(authorID)
		if err != nil {
			respondWithError(w, 500, "Unable to retrieve chirps.")
			return
		}
		respondWithJSON(w, 200, chirps)
		return
	}

	order := r.URL.Query().Get("sort")
	sortOrder := "asc"
	if order == "desc" {
		sortOrder = "desc"
	} else if order != "asc" {
		respondWithError(w, http.StatusBadRequest, "Invalid parameter for sort")
	}

	chirps, err := cfg.db.GetChirps(sortOrder)
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
