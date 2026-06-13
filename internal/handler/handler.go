package handler

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
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
	r.Get("/{id}", h.expandHandler)

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
	})
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

	if parsedURL, err := url.ParseRequestURI(originalURL); err != nil || parsedURL.Scheme == "" {
		if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
			originalURL = "http://" + originalURL
		}

		parsedURL, err = url.ParseRequestURI(originalURL)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
	} else {
		if parsedURL.Host == "" {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
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
