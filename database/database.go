package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

const DB_PATH string = "database.json"

type DB struct {
	path string
	mux  *sync.Mutex
}

type DBStructure struct {
	Data struct {
		Users  Users  `json:"users"`
		Chirps Chirps `json:"chirps"`
	} `json:"data"`
}

type Chirp struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}

type Chirps struct {
	Chirps map[int]Chirp `json:"chirps"`
}

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Users struct {
	Users map[int]User
}

func NewDB(path string) (*DB, error) {
	mux := sync.Mutex{}
	db := &DB{path, &mux}
	err := db.ensureDB()
	if err != nil {
		log.Println(err)
		return &DB{}, err
	}
	return db, nil

}

func (db *DB) ensureDB() error {
	if _, err := os.Stat(db.path); os.IsNotExist(err) {
		err := os.WriteFile(db.path, []byte(""), 0666)
		if err != nil {
			return fmt.Errorf("Error creating DB: %s", err)
		}
		log.Printf("Created database at %s", db.path)
		return nil
	}
	return nil
}

func (db *DB) loadDB() (DBStructure, error) {
	defer db.mux.Unlock()
	db.mux.Lock()
	dat, err := os.ReadFile(db.path)
	if err != nil {
		return DBStructure{}, fmt.Errorf("Unable to read DB file: %s", err)
	}
	if len(dat) == 0 {
		dbStructure := DBStructure{}
		dbStructure.Data.Users.Users = make(map[int]User)
		dbStructure.Data.Chirps.Chirps = make(map[int]Chirp)

		return dbStructure, nil
	}
	dbStructure := DBStructure{}
	err = json.Unmarshal(dat, &dbStructure)
	if err != nil {
		return DBStructure{}, fmt.Errorf("Unable to unmarshal data from DBfile: %s", err)
	}
	return dbStructure, nil
}

func (db *DB) writeDB(dbStructure DBStructure) error {
	defer db.mux.Unlock()
	db.mux.Lock()
	dat, err := json.Marshal(dbStructure)
	if err != nil {
		return fmt.Errorf("Unable to write to DB: %s", err)
	}
	err = os.WriteFile(db.path, dat, 0666)
	return nil
}

func (db *DB) CreateChirp(body string) (Chirp, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}
	id := len(dbStructure.Data.Chirps.Chirps) + 1
	chirp := Chirp{id, body}
	dbStructure.Data.Chirps.Chirps[id] = chirp
	err = db.writeDB(dbStructure)
	if err != nil {
		return Chirp{}, err
	}
	return chirp, nil
}

func (db *DB) GetChirps() ([]Chirp, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return []Chirp{}, err
	}
	chirps := []Chirp{}
	for _, v := range dbStructure.Data.Chirps.Chirps {
		chirps = append(chirps, v)
	}
	sort.Slice(chirps, func(i, j int) bool { return chirps[i].ID < chirps[j].ID })
	return chirps, nil

}

func (db *DB) GetChirp(chirpID int) (Chirp, error) {
	chirps, err := db.GetChirps()
	if err != nil {
		return Chirp{}, err
	}

	if chirpID > len(chirps) {
		return Chirp{}, fmt.Errorf("Chirp with chirpID %d does not exist.", chirpID)
	}
	chirp := chirps[chirpID-1]
	if chirp.ID != chirpID {
		return Chirp{}, errors.New("Internal error in retrieving chirp.")
	}

	return chirp, nil
}

func (db *DB) CreateUser(email string, password string) (User, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}
	users := dbStructure.Data.Users.Users
	id := len(users) + 1
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}

	user := User{
		ID:       id,
		Email:    email,
		Password: string(hashedPwd),
	}
	for _, u := range users {
		if u.Email == email {
			return User{}, errors.New("User already exists.")
		}
	}
	dbStructure.Data.Users.Users[id] = user
	err = db.writeDB(dbStructure)
	if err != nil {
		log.Println(err.Error())
		return User{}, err
	}

	return user, nil
}

func (db *DB) GetUser(email string) (User, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	users := dbStructure.Data.Users.Users
	for _, user := range users {
		if user.Email == email {
			return user, nil
		}
	}
	return User{}, fmt.Errorf("User not found: %s", email)
}
