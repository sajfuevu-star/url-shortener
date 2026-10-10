package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"url/handler"
	"url/repository"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")

	db, err := repository.NewPostgres(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	log.Println("PostgreSQL подключен")

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
