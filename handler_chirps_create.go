package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/golang-jwt/jwt/v4"
)

func (cfg *apiConfig) handlerCreateChirps(w http.ResponseWriter, r *http.Request) {
	tokenString, err := getBearerTokenFromHeader(r)
	if err != nil {
		respondWithError(w, 401, err.Error())
	}
	fmt.Println(tokenString)
	fmt.Println(cfg.jwtSecret)
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.jwtSecret), nil
	})
	if err != nil {
		respondWithError(w, 401, fmt.Sprintf("Cannot parse JWT token: %s", err.Error()))
		return
	}

	claims := token.Claims.(*jwt.RegisteredClaims)
	id := claims.Subject

	authorID, err := strconv.Atoi(id)
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}

	// get the body of the request
	type parameters struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	// decode body
	if err != nil {
		log.Printf("Error decoding parameters %s\n", err)
		respondWithError(w, 500, "Error decoding parameters.")
		return
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

	c, err := cfg.db.CreateChirp(cleanedBody, authorID)
	if err != nil {
		log.Printf("Error saving chirp.")
		errMsg := fmt.Sprintf("Error saving chirp: %s", err)
		respondWithError(w, 500, errMsg)
		return
	}
	respondWithJSON(w, 201, c)
	return
}
