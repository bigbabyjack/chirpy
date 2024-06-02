package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/bigbabyjack/chirpy/database"
)

type apiConfig struct {
	fileserverHits int
}

const dbPath string = "database.json"
const port string = "8080"
const filepathRoot string = "/"

func main() {
	cfg := &apiConfig{
		fileserverHits: 0,
	}

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	db, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Error starting database: %s", err)
	}

	mux.Handle("/app/*", cfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		html := fmt.Sprintf(`<html>
			<body>
				<h1>Welcome, Chirpy Admin</h1>
				<p>Chirpy has been visited %d times!</p>
			</body>

			</html>`,
			cfg.fileserverHits)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})
	mux.HandleFunc("/api/reset", cfg.handlerReset)
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
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

		c, err := db.CreateChirp(cleanedBody)
		if err != nil {
			log.Printf("Error saving chirp.")
			errMsg := fmt.Sprintf("Error saving chirp: %s", err)
			respondWithError(w, 500, errMsg)
			return
		}
		respondWithJSON(w, 201, c)
		return
	})

	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		chirps, err := db.GetChirps()
		if err != nil {
			respondWithError(w, 500, "Unable to retrieve chirps.")
		}
		respondWithJSON(w, 200, chirps)

	})

	mux.HandleFunc("GET /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {
		chirpID, err := strconv.Atoi(r.PathValue("chirpID"))
		if err != nil {
			respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid chirpID %v", chirpID))
			return
		}
		chirp, err := db.GetChirp(chirpID)
		if err != nil {
			respondWithError(w, 404, fmt.Sprintf("Chirp with ID %v not found", chirpID))
			return
		}
		respondWithJSON(w, 200, chirp)
		return

	})

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())

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
