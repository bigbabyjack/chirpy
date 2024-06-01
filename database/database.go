package database

import (
	"encoding/json"
	"log"
	"os"
)

func writeToDisk(payload interface{}) error {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error writing to disk, %s: %s", DB_PATH, err)
		return err
	}
	err = os.WriteFile(DB_PATH, []byte(dat), os.ModePerm)
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
