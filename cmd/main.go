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
	if err := db.Ping(ctx); err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	repo := repository.NewLinkRepository(db)

	log.Println("PostgreSQL подключен")

	mux := http.NewServeMux()

	manager := handler.NewManager(repo)

	mux.HandleFunc("POST /shorten", manager.CreateShortLink)
	mux.HandleFunc("GET /r/{code}", manager.Redirect)
	mux.HandleFunc("DELETE /links/{code}", manager.DeleteLink)
	mux.HandleFunc("GET /links", manager.GetAllLinks)
	mux.HandleFunc("GET /stats/{code}", manager.GetStats)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Fatal(server.ListenAndServe())
}
