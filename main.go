package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bigbabyjack/chirpy/database"
	"github.com/golang-jwt/jwt/v4"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type apiConfig struct {
	fileserverHits int
	db             *database.DB
	jwtSecret      string
}

const dbPath string = "database.json"
const port string = "8080"
const filepathRoot string = "/"

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatalf("JWT_SECRET not found in .env file")
	}
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
		jwtSecret:      jwtSecret,
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
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Hello from the delete endpoint")
		tokenString, err := getBearerTokenFromHeader(r)
		if err != nil {
			respondWithError(w, 403, "Cannot parse JWT token")
			return
		}
		token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.jwtSecret), nil
		})
		if err != nil {
			respondWithError(w, 403, "Cannot parse JWT token")
			return
		}

		claims := token.Claims.(*jwt.RegisteredClaims)
		authorID, err := strconv.Atoi(claims.Subject)
		if err != nil {
			respondWithError(w, 403, "Unauthorized.")
			return
		}

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
		if authorID == chirp.AuthorID {
			err := cfg.db.DeleteChirp(chirp.ID)
			if err != nil {
				respondWithError(w, 500, err.Error())
				return
			}
			fmt.Println("Deleted")
			respondWithJSON(w, 204, struct{}{})
			return
		}
		respondWithError(w, 403, "Unauthorized.")
		return
	})

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
		hashedPwd, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
		if err != nil {
			respondWithError(w, 500, "Password must be between 5 and 12 characters.")
		}
		user, err := cfg.db.CreateUser(params.Email, string(hashedPwd))
		if err != nil {
			respondWithError(w, 500, "Error creating user.")
		}
		respondWithJSON(w, 201, UserResponse{
			user.ID,
			user.Email,
			user.IsChirpyRed,
		})
	})

	mux.HandleFunc("PUT /api/users", func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := getBearerTokenFromHeader(r)
		if err != nil {
			respondWithError(w, 401, "Cannot parse JWT token")
			return
		}
		token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.jwtSecret), nil
		})
		if err != nil {
			respondWithError(w, 401, "Cannot parse JWT token")
			return
		}

		claims := token.Claims.(*jwt.RegisteredClaims)
		id := claims.Subject

		params := database.User{}
		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&params)
		if err != nil {
			respondWithError(w, 500, err.Error())
			return
		}

		ID, err := strconv.Atoi(id)
		if err != nil {
			respondWithError(w, 500, err.Error())
			return
		}
		hashedPwd, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
		params.Password = string(hashedPwd)
		if err != nil {
			respondWithError(w, 500, "Password must be between 5 and 12 characters.")
			return
		}

		user, err := cfg.db.UpdateUser(ID, params)
		if err != nil {
			respondWithError(w, 500, err.Error())
			return
		}
		respondWithJSON(w, 200, UserResponse{
			user.ID,
			params.Email,
			user.IsChirpyRed,
		})

	})

	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Email            string `json:"email"`
			Password         string `json:"password"`
			ExpiresInSeconds *int64 `json:"expires_in_seconds"`
		}
		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			respondWithError(w, 500, "Unable to parse email and password")
			return
		}
		user, err := db.GetUser(params.Email)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		// compare password hash
		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(params.Password))
		if err != nil {
			respondWithError(w, 401, "Invalid username and password combination.")
			return
		}

		var expiresInSeconds int64
		if params.ExpiresInSeconds != nil {
			params.ExpiresInSeconds = &expiresInSeconds
		} else {
			expiresInSeconds = int64(1 * time.Hour.Seconds())
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
			Issuer:    "chirpy",
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Duration(expiresInSeconds) * time.Second)),
			Subject:   strconv.Itoa(user.ID),
		})

		signedToken, err := token.SignedString([]byte(cfg.jwtSecret))
		if err != nil {
			log.Println(err.Error())
			respondWithError(w, 500, "Unable to get JWT Token")
			return
		}
		b := make([]byte, 32)
		_, err = rand.Read(b)
		if err != nil {
			respondWithError(w, 500, "Internal Error")
			return
		}
		refreshToken := hex.EncodeToString(b)
		err = cfg.db.UpdateRefreshToken(user.ID, refreshToken)
		if err != nil {
			respondWithError(w, 500, "Internal Error")
			return
		}

		respondWithJSON(w, 200, UserResponseWithToken{
			user.ID,
			user.Email,
			signedToken,
			refreshToken,
			user.IsChirpyRed,
		})
	})

	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
		refreshToken, err := getBearerTokenFromHeader(r)
		if err != nil {
			respondWithError(w, 401, err.Error())
		}
		u, err := cfg.db.VerifyRefreshToken(refreshToken)
		if err != nil {
			respondWithError(w, 401, "Unauthorized user.")
			return
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
			Issuer:    "chirpy",
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Duration(time.Hour.Seconds()) * time.Second)),
			Subject:   strconv.Itoa(u.ID),
		})

		signedToken, err := token.SignedString([]byte(cfg.jwtSecret))
		respondWithJSON(w, 200, struct {
			Token string `json:"token"`
		}{signedToken})
	})

	mux.HandleFunc("POST /api/revoke", func(w http.ResponseWriter, r *http.Request) {
		refreshToken, err := getBearerTokenFromHeader(r)
		if err != nil {
			respondWithError(w, 401, err.Error())
			return
		}
		err = cfg.db.RevokeRefreshToken(refreshToken)
		if err != nil {
			respondWithError(w, 401, "Invalid token")
			return
		}
		respondWithJSON(w, 204, struct{}{})
	})

	mux.HandleFunc("POST /api/polka/webhooks", func(w http.ResponseWriter, r *http.Request) {
		type polkaHook struct {
			Event string         `json:"event"`
			Data  map[string]int `json:"data"`
		}

		request := &polkaHook{}
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(request)
		if err != nil {
			respondWithError(w, 404, err.Error())
			return
		}
		if request.Event != "user.upgraded" {
			respondWithError(w, 204, "")
			return
		}
		userID, ok := request.Data["user_id"]
		if !ok {
			respondWithError(w, 404, "")
			return
		}
		user, err := cfg.db.GetUserByID(userID)
		if err != nil {
			respondWithError(w, 404, err.Error())
			return
		}
		user.IsChirpyRed = true
		_, err = cfg.db.UpdateUser(userID, user)
		if err != nil {
			respondWithError(w, 500, err.Error())
			return
		}
		respondWithJSON(w, 204, fmt.Sprintf("User %d upgraded.", userID))

	})

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())

}

type UserResponseWithToken struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	JWTToken     string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	IsChirpyRed  bool   `json:"is_chirpy_red"`
}

type UserResponse struct {
	ID          int    `json:"id"`
	Email       string `json:"email"`
	IsChirpyRed bool   `json:"is_chirpy_red"`
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

func getBearerTokenFromHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("Unable to parse bearer token")
	}

	refreshToken := strings.TrimPrefix(authHeader, "Bearer ")
	return refreshToken, nil
}
