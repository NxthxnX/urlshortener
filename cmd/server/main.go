package main

import (
	"net/http"

	"github.com/NxthxnX/urlshortener/internal/handler"
	"github.com/NxthxnX/urlshortener/internal/logger"
	"github.com/NxthxnX/urlshortener/internal/repository"
	"github.com/NxthxnX/urlshortener/internal/service"
	"github.com/go-chi/chi/v5"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	if err := logger.Initialize("info"); err != nil {
		return err
	}

	parseFlags()

	repo := repository.NewMemoryRepository()
	svc := service.NewShortenerService(repo)
	h := handler.NewHandler(svc, string(options.baseURL))

	r := chi.NewRouter()
	r.Use(logger.WithLogging)
	h.RegisterRoutes(r)

	return http.ListenAndServe(string(options.servAddr), r)
}
