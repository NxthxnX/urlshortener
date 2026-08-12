package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NxthxnX/urlshortener/internal/config"
	"github.com/NxthxnX/urlshortener/internal/config/db"
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

	store, err := db.Open(cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	defer store.Close()

	if err := store.Migrate(); err != nil {
		return err
	}

	repo, err := repository.New(repository.Config{
		FileStoragePath: cfg.FileStoragePath,
		DatabaseDSN:     cfg.DatabaseDSN,
		DB:              store.DB(),
	})
	if err != nil {
		return err
	}

	svc := service.NewShortenerService(repo)
	h := handler.NewHandler(svc, cfg.BaseURL)

	r := chi.NewRouter()
	r.Use(logger.WithLogging, middleware.WithEncoding)
	r.Get("/ping", handler.PingHandler(store))
	h.RegisterRoutes(r)

	server := &http.Server{
		Addr:    cfg.ServAddr,
		Handler: r,
	}

	errCh := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-quit:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	}
}
