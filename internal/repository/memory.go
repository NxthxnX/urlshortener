package repository

import (
	"sync"

	"github.com/NxthxnX/urlshortener/internal/model"
)

var _ Repository = (*MemoryRepository)(nil)

// MemoryRepository stores URL mappings in memory.
type MemoryRepository struct {
	mu   sync.RWMutex
	urls map[string]string // id -> originalURL
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		urls: make(map[string]string),
	}
}

// Save stores a mapping between id and originalURL.
func (r *MemoryRepository) Save(id, originalURL string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.urls[id] = originalURL
}

// SaveBatch stores multiple URL mappings in memory.
func (r *MemoryRepository) SaveBatch(pairs []model.URLPair) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range pairs {
		r.urls[p.ShortURL] = p.OriginalURL
	}
	return nil
}

// FindByID retrieves the original URL by its shortened ID.
// Returns the original URL and a boolean indicating if found.
func (r *MemoryRepository) FindByID(id string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	originalURL, ok := r.urls[id]
	return originalURL, ok
}
