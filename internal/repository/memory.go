package repository

import (
	"fmt"
	"sync"

	"github.com/NxthxnX/urlshortener/internal/model"
)

var _ Repository = (*MemoryRepository)(nil)

// MemoryRepository stores URL mappings in memory.
type MemoryRepository struct {
	mu          sync.RWMutex
	urls        map[string]string // shortURL -> originalURL
	origToShort map[string]string // originalURL -> shortURL
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		urls:        make(map[string]string),
		origToShort: make(map[string]string),
	}
}

// Save stores a mapping between id and originalURL.
// If originalURL already exists, returns the existing short URL and
// an error wrapping ErrOriginalURLConflict. If short_url collides,
// returns an error wrapping ErrShortURLConflict.
func (r *MemoryRepository) Save(id, originalURL string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existingID, ok := r.origToShort[originalURL]; ok {
		return existingID, fmt.Errorf("%w: original URL already exists", ErrOriginalURLConflict)
	}

	if _, exists := r.urls[id]; exists {
		return "", fmt.Errorf("%w: short URL already exists", ErrShortURLConflict)
	}

	r.urls[id] = originalURL
	r.origToShort[originalURL] = id
	return id, nil
}

// SaveBatch stores multiple URL mappings in memory.
// If any original URL already exists, the existing short URL is used
// and the returned error wraps ErrOriginalURLConflict.
func (r *MemoryRepository) SaveBatch(pairs []model.URLPair) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ids := make([]string, 0, len(pairs))
	hasConflict := false

	for _, p := range pairs {
		if existingID, ok := r.origToShort[p.OriginalURL]; ok {
			ids = append(ids, existingID)
			hasConflict = true
			continue
		}

		if _, exists := r.urls[p.ShortURL]; exists {
			return nil, fmt.Errorf("%w: short URL already exists", ErrShortURLConflict)
		}

		r.urls[p.ShortURL] = p.OriginalURL
		r.origToShort[p.OriginalURL] = p.ShortURL
		ids = append(ids, p.ShortURL)
	}

	if hasConflict {
		return ids, fmt.Errorf("%w: some original URLs already exist", ErrOriginalURLConflict)
	}

	return ids, nil
}

// FindByID retrieves the original URL by its shortened ID.
// Returns the original URL and a boolean indicating if found.
func (r *MemoryRepository) FindByID(id string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	originalURL, ok := r.urls[id]
	return originalURL, ok
}

// Clear removes all stored URL mappings.
func (r *MemoryRepository) Clear() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.urls = make(map[string]string)
	r.origToShort = make(map[string]string)
	return nil
}
