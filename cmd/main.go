package main

import (
	"log"
	"net/http"

	"url/handler"
)

func main() {
	mux := http.NewServeMux()

	manager := handler.NewManager()

	mux.HandleFunc("POST /shorten", manager.CreateShortLink)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Fatal(server.ListenAndServe())
}
