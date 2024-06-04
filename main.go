package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/bigbabyjack/chirpy/database"
	"golang.org/x/crypto/bcrypt"
)

type apiConfig struct {
	fileserverHits int
	db             *database.DB
}

const dbPath string = "database.json"
const port string = "8080"
const filepathRoot string = "/"

func main() {

	dbg := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()
	if *dbg {
		err := os.Remove(dbPath)
		if err != nil {
			log.Fatalf("Unable to delete database in debug mode: %s", err.Error())
			return
		}
	}

	db, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Error starting database: %s", err)
	}
	cfg := &apiConfig{
		fileserverHits: 0,
		db:             db,
	}

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
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

	mux.HandleFunc("POST /api/chirps", cfg.handlerCreateChirps)
	mux.HandleFunc("GET /api/chirps", cfg.handlerGetChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", cfg.handlerGetChirp)

	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			respondWithError(w, 500, "Error decoding parameters.")
			return
		}
		err = verifyPasswordCreation(params.Password)
		if err != nil {
			respondWithError(w, 500, "Password must be between 5 and 12 characters.")
		}
		user, err := cfg.db.CreateUser(params.Email, params.Password)
		if err != nil {
			respondWithError(w, 500, "Error creating user.")
		}
		respondWithJSON(w, 201, UserResponse{
			user.ID,
			user.Email,
		})
	})

	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			respondWithError(w, 500, "Unable to parse email and password")
		}
		// TODO:
		// get user by email
		user, err := db.GetUser(params.Email)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
		}
		// compare password hash
		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(params.Password))
		if err != nil {
			respondWithError(w, 401, "Invalid username and password combination.")
		}
		respondWithJSON(w, 200, UserResponse{
			user.ID,
			user.Email,
		})
	})

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())

}

type UserResponse struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}

func verifyPasswordCreation(p string) error {
	if len(p) > 12 || len(p) < 5 {
		return fmt.Errorf("Password must be between 5 and 12 characters")
	}

	return nil
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
