package handler

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/NxthxnX/urlshortener/internal/myjson"
	"github.com/go-chi/chi/v5"
	"github.com/mailru/easyjson"
)

// Shortener defines the interface for URL shortening operations.
type Shortener interface {
	Shorten(originalURL string) (string, error)
	Expand(id string) (string, bool)
}

// Handler handles HTTP requests for the URL shortener.
type Handler struct {
	shortener Shortener
	baseURL   string
}

// New creates a new Handler.
func NewHandler(s Shortener, baseURL string) *Handler {
	return &Handler{
		shortener: s,
		baseURL:   baseURL,
	}
}

// RegisterRoutes registers all routes with the chi router.
func (h *Handler) RegisterRoutes(r *chi.Mux) {
	r.Post("/", h.shortenHandler)
	r.Post("/api/shorten", h.apiShortenHandler)
	r.Get("/{id}", h.expandHandler)

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
	})
}

// parseURL parses a raw URL and returns an error if it's invalid.
func parseURL(rawURL *string) error {
	if parsedURL, err := url.ParseRequestURI(*rawURL); err != nil || parsedURL.Scheme == "" {
		if !strings.HasPrefix(*rawURL, "http://") && !strings.HasPrefix(*rawURL, "https://") {
			*rawURL = "http://" + *rawURL
		}

		parsedURL, err = url.ParseRequestURI(*rawURL)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			return errors.New("invalid URL")
		}
	} else {
		if parsedURL.Host == "" {
			return errors.New("invalid URL")
		}
	}

	return nil
}

// shortenHandler handles POST / requests to shorten a URL.
func (h *Handler) shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Cannot read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(w, "Empty URL", http.StatusBadRequest)
		return
	}

	if err := parseURL(&originalURL); err != nil {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	id, err := h.shortener.Shorten(originalURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	shortURL := h.baseURL + "/" + id

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(shortURL)))
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

// shortenHandler handles POST /api/shorten JSON requests to shorten a URL.
func (h *Handler) apiShortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/shorten" {
		http.Error(w, "Not found", http.StatusBadRequest)
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Cannot read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	req := myjson.ApiShortenRequest{}

	if err := easyjson.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	originalURL := strings.TrimSpace(req.URL)
	if originalURL == "" {
		http.Error(w, "Empty URL", http.StatusBadRequest)
		return
	}

	if err := parseURL(&originalURL); err != nil {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	id, err := h.shortener.Shorten(originalURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	shortURL := h.baseURL + "/" + id

	resp := myjson.ApiShortenResponse{Result: shortURL}
	jsonBody, err := easyjson.Marshal(resp)
	if err != nil {
		http.Error(w, "Error encoding JSON", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonBody)))
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
		http.Error(w, "Not found", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}
