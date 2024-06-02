package database

import (
	"encoding/json"
	"os"
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
		return &DB{}, err
	}
	return db, nil

}

func (db *DB) ensureDB() error {
	if _, err := os.Stat(db.path); err != nil {
		if os.IsNotExist(err) {
			file, err := os.OpenFile(db.path, os.O_CREATE|os.O_WRONLY, 0666)
			defer file.Close()
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func (db *DB) loadDB() (DBStructure, error) {
	dat, err := os.ReadFile(db.path)
	if err != nil {
		return DBStructure{}, err
	}
	dbStructure := DBStructure{}
	err = json.Unmarshal(dat, &dbStructure)
	if err != nil {
		return DBStructure{}, err
	}
	return dbStructure, nil
}

func (db *DB) writeDB(dbStructure DBStructure) error {
	dat, err := json.Marshal(dbStructure)
	if err != nil {
		return err
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
	return chirps, nil

}
