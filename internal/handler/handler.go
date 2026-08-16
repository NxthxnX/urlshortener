package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/NxthxnX/urlshortener/internal/myjson"
	"github.com/NxthxnX/urlshortener/internal/repository"
	"github.com/NxthxnX/urlshortener/internal/urlutils"
	"github.com/go-chi/chi/v5"
	"github.com/mailru/easyjson"
)

var (
	errEmptyURL   = errors.New("empty URL")
	errInvalidURL = errors.New("invalid URL")
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_handler.go -package=mocks

// Shortener defines the interface for URL shortening operations.
type Shortener interface {
	Shorten(originalURL string) (string, error)
	ShortenBatch(originalURLs []string) ([]string, error)
	Expand(id string) (string, bool)
}

// Pinger checks database connectivity.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Handler handles HTTP requests for the URL shortener.
type Handler struct {
	shortener Shortener
	baseURL   string
}

// NewHandler creates a new Handler.
func NewHandler(s Shortener, baseURL string) *Handler {
	return &Handler{
		shortener: s,
		baseURL:   strings.TrimSuffix(baseURL, "/"),
	}
}

// PingHandler returns a handler that checks the database connection via GET /ping.
// Register it on the router before parameterized routes like /{id}.
func PingHandler(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			http.Error(w, "database is not configured", http.StatusInternalServerError)
			return
		}

		if err := db.Ping(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// RegisterRoutes registers URL shortener routes with the chi router.
func (h *Handler) RegisterRoutes(r *chi.Mux) {
	r.Post("/", h.shortenHandler)
	r.Post("/api/shorten", h.apiShortenHandler)
	r.Post("/api/shorten/batch", h.apiShortenBatchHandler)
	r.Get("/{id}", h.expandHandler)
}

func (h *Handler) buildShortURL(id string) string {
	return h.baseURL + "/" + id
}

// normalizeURL trims whitespace, validates and normalizes a raw URL.
func normalizeURL(rawURL string) (string, error) {
	originalURL := strings.TrimSpace(rawURL)
	if originalURL == "" {
		return "", errEmptyURL
	}

	normalized, err := urlutils.Normalize(originalURL)
	if err != nil {
		return "", errInvalidURL
	}

	return normalized, nil
}

func (h *Handler) shortenURL(rawURL string) (string, error) {
	normalized, err := normalizeURL(rawURL)
	if err != nil {
		return "", err
	}

	id, err := h.shortener.Shorten(normalized)
	if err != nil {
		if errors.Is(err, repository.ErrOriginalURLConflict) {
			return h.buildShortURL(id), err
		}
		return "", err
	}

	return h.buildShortURL(id), nil
}

func writeShortenError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errEmptyURL):
		http.Error(w, "Empty URL", http.StatusBadRequest)
	case errors.Is(err, errInvalidURL):
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
	case errors.Is(err, repository.ErrShortURLConflict):
		http.Error(w, "Short URL conflict", http.StatusInternalServerError)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// shortenHandler handles POST / requests to shorten a URL.
func (h *Handler) shortenHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Cannot read body", http.StatusBadRequest)
		return
	}

	shortURL, err := h.shortenURL(string(body))
	if err != nil {
		if errors.Is(err, repository.ErrOriginalURLConflict) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(shortURL))
			return
		}
		writeShortenError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

// apiShortenHandler handles POST /api/shorten JSON requests to shorten a URL.
func (h *Handler) apiShortenHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		http.Error(w, "Invalid Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Cannot read body", http.StatusBadRequest)
		return
	}

	req := myjson.APIShortenRequest{}

	if err := easyjson.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	shortURL, err := h.shortenURL(req.URL)
	if err != nil {
		if errors.Is(err, repository.ErrOriginalURLConflict) {
			resp := myjson.APIShortenResponse{Result: shortURL}
			jsonBody, err := easyjson.Marshal(resp)
			if err != nil {
				http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			w.Write(jsonBody)
			return
		}
		writeShortenError(w, err)
		return
	}

	resp := myjson.APIShortenResponse{Result: shortURL}
	jsonBody, err := easyjson.Marshal(resp)
	if err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(jsonBody)
}

// apiShortenBatchHandler handles POST /api/shorten/batch
// JSON requests to shorten a batch of URLs.
func (h *Handler) apiShortenBatchHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		http.Error(w, "Invalid Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Cannot read body", http.StatusBadRequest)
		return
	}

	var req []myjson.APIShortenBatchRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req) == 0 {
		http.Error(w, "Empty batch", http.StatusBadRequest)
		return
	}

	seen := make(map[string]struct{})
	for _, item := range req {
		if _, exists := seen[item.CorrelationID]; exists {
			http.Error(w, "Duplicate correlation_id", http.StatusBadRequest)
			return
		}
		seen[item.CorrelationID] = struct{}{}
	}

	normalizedURLs := make([]string, 0, len(req))
	for _, item := range req {
		normalized, err := normalizeURL(item.OriginalURL)
		if err != nil {
			writeShortenError(w, err)
			return
		}
		normalizedURLs = append(normalizedURLs, normalized)
	}

	ids, err := h.shortener.ShortenBatch(normalizedURLs)
	if err != nil {
		if errors.Is(err, repository.ErrOriginalURLConflict) {
			resp := make([]myjson.APIShortenBatchResponse, 0, len(req))
			for i, item := range req {
				resp = append(resp, myjson.APIShortenBatchResponse{
					CorrelationID: item.CorrelationID,
					ShortURL:      h.buildShortURL(ids[i]),
				})
			}

			jsonBody, err := json.Marshal(resp)
			if err != nil {
				http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			w.Write(jsonBody)
			return
		}
		writeShortenError(w, err)
		return
	}

	resp := make([]myjson.APIShortenBatchResponse, 0, len(req))
	for i, item := range req {
		resp = append(resp, myjson.APIShortenBatchResponse{
			CorrelationID: item.CorrelationID,
			ShortURL:      h.buildShortURL(ids[i]),
		})
	}

	jsonBody, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(jsonBody)
}

// expandHandler handles GET /{id} requests to expand a shortened URL.
func (h *Handler) expandHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	originalURL, ok := h.shortener.Expand(id)
	if !ok {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}
