package service

import (
	"crypto/rand"
	"math/big"

	"github.com/NxthxnX/urlshortener/internal/repository"
)

const (
	idLength   = 8
	idAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// ShortenerService provides business logic for URL shortening.
type ShortenerService struct {
	repo repository.Repository
}

// NewShortenerService creates a new ShortenerService.
func NewShortenerService(repo repository.Repository) *ShortenerService {
	return &ShortenerService{repo: repo}
}

// Shorten generates a short ID for the given URL and stores it.
// Returns the generated ID and an error indicating if generation has failed.
func (s *ShortenerService) Shorten(originalURL string) (string, error) {
	id, err := generateID()
	if err != nil {
		return "", err
	}

	s.repo.Save(id, originalURL)
	return id, nil
}

// Expand retrieves the original URL for the given short ID.
// Returns the original URL and a boolean indicating if found.
func (s *ShortenerService) Expand(id string) (string, bool) {
	record, ok := s.repo.FindByID(id)
	if !ok {
		return "", false
	}
	return record, true
}

// generateID creates a random alphanumeric string of idLength characters.
func generateID() (string, error) {
	chars := make([]byte, idLength)
	for i := range chars {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(idAlphabet))))
		if err != nil {
			return "", err
		}
		chars[i] = idAlphabet[idx.Int64()]
	}
	return string(chars), nil
}
