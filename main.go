package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
)

type apiConfig struct {
	countChirpsOnDisk int
	fileserverHits    int
}

const DB_PATH string = "database.json"

func main() {
	count, err := getCountChirps()
	if err != nil {
		log.Printf("Unable to get chirp count: %s", err.Error())
	}
	cfg := &apiConfig{
		countChirpsOnDisk: count,
		fileserverHits:    0,
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

		c := chirp{
			ID:   0,
			Body: cleanedBody,
		}
		err = writeToDisk(c)
		if err != nil {
			log.Printf("Error saving chirp.")
			respondWithError(w, 500, "Error saving chirp.")
			return
		}
		respondWithJSON(w, 201, c)
		return
	})

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())

}

func writeToDisk(payload interface{}) error {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error writing to disk, %s: %s", DB_PATH, err)
		return err
	}
	err = os.WriteFile(DB_PATH, []byte(dat), os.ModeAppend)
	if err != nil {
		log.Printf("Error writing to disk, %s: %s", DB_PATH, err)
		return err
	}
	return nil
}

type chirp struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}

type chirps struct {
	Chirps []chirp `json:"chirps"`
}

func readSavedChirps() ([]byte, error) {
	dat, err := os.ReadFile(DB_PATH)
	if err != nil {
		return nil, err
	}

	return dat, nil
}

func getCountChirps() (int, error) {
	dat, err := readSavedChirps()
	if err != nil {
		return 0, err
	}
	chirps := chirps{}
	err = json.Unmarshal(dat, &chirps)
	if err != nil {
		return 0, err
	}
	return len(chirps.Chirps), nil
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
