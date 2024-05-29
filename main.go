package main

import (
	"log"
	"net/http"
)

const port string = "8080"
const filepathRoot string = "/"

func main() {
	mux := http.NewServeMux()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	mux.Handle(filepathRoot, http.FileServer(http.Dir(".")))

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())

}
