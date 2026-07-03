package handler

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/NxthxnX/urlshortener/internal/myjson"
	"github.com/NxthxnX/urlshortener/internal/urlutils"
	"github.com/go-chi/chi/v5"
	"github.com/mailru/easyjson"
)

var (
	errEmptyURL   = errors.New("empty URL")
	errInvalidURL = errors.New("invalid URL")
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

// NewHandler creates a new Handler.
func NewHandler(s Shortener, baseURL string) *Handler {
	return &Handler{
		shortener: s,
		baseURL:   strings.TrimSuffix(baseURL, "/"),
	}
}

// RegisterRoutes registers all routes with the chi router.
func (h *Handler) RegisterRoutes(r *chi.Mux) {
	r.Post("/", h.shortenHandler)
	r.Post("/api/shorten", h.apiShortenHandler)
	r.Get("/{id}", h.expandHandler)

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
}

func (h *Handler) buildShortURL(id string) string {
	return h.baseURL + "/" + id
}

func (h *Handler) shortenURL(rawURL string) (string, error) {
	originalURL := strings.TrimSpace(rawURL)
	if originalURL == "" {
		return "", errEmptyURL
	}

	normalized, err := urlutils.Normalize(originalURL)
	if err != nil {
		return "", errInvalidURL
	}

	id, err := h.shortener.Shorten(normalized)
	if err != nil {
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
