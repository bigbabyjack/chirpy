package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
)

const DB_PATH string = "database.json"

type Chirp struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}

type DB struct {
	path string
	mux  *sync.Mutex
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
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
		return DBStructure{Chirps: make(map[int]Chirp)}, nil
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
	id := len(dbStructure.Chirps) + 1
	chirp := Chirp{id, body}
	dbStructure.Chirps[id] = chirp
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
	for _, v := range dbStructure.Chirps {
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
