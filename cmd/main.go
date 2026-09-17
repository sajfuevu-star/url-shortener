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
	mux.HandleFunc("GET /r/{code}", manager.Redirect)
	mux.HandleFunc("DELETE /links/{code}", manager.DeleteLink)
	mux.HandleFunc("GET /links", manager.GetAllLinks)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Fatal(server.ListenAndServe())
}
