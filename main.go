package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
)

type apiConfig struct {
	fileserverHits int
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Print("Increasing count.\n")
		cfg.fileserverHits++
		log.Printf("Count of hits: %v", cfg.fileserverHits)
		next.ServeHTTP(w, r) // call the next handler
	})
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits = 0
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}

func main() {
	cfg := &apiConfig{
		fileserverHits: 0,
	}
	const port string = "8080"
	const filepathRoot string = "/"

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	mux.Handle("/app/*", cfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		html := fmt.Sprintf(`<html>

<body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
</body>

</html>`, cfg.fileserverHits)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})
	mux.HandleFunc("/api/reset", cfg.handlerReset)
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
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
