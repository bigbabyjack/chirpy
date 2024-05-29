package main

import (
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	srv := http.Server{
		Addr:    "localhost:8080",
		Handler: mux,
	}
	srv.ListenAndServe()
}
