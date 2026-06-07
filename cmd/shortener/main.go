package main

import (
	"net/http"

	"github.com/NxthxnX/urlshortener/internal/handler"
	"github.com/NxthxnX/urlshortener/internal/repository"
	"github.com/NxthxnX/urlshortener/internal/service"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortenerService(repo)
	h := handler.NewHandler(svc)

	return http.ListenAndServe(`:8080`, h)
}
