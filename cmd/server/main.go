package main

import (
	"net/http"

	"github.com/NxthxnX/urlshortener/internal/config"
	"github.com/NxthxnX/urlshortener/internal/handler"
	"github.com/NxthxnX/urlshortener/internal/logger"
	"github.com/NxthxnX/urlshortener/internal/middleware"
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

	cfg := config.ParseFlags()

	repo, err := repository.New(repository.Config{
		FileStoragePath: cfg.FileStoragePath,
	})
	if err != nil {
		return err
	}

	svc := service.NewShortenerService(repo)
	h := handler.NewHandler(svc, cfg.BaseURL)

	r := chi.NewRouter()
	r.Use(logger.WithLogging, middleware.WithEncoding)
	h.RegisterRoutes(r)

	return http.ListenAndServe(cfg.ServAddr, r)
}
