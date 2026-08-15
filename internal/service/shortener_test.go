package service

import (
	"testing"

	"github.com/NxthxnX/urlshortener/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortenerService_ShortenBatch(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := NewShortenerService(repo)

	urls := []string{"http://example.com", "https://yandex.ru"}
	ids, err := svc.ShortenBatch(urls)
	require.NoError(t, err)
	require.Len(t, ids, len(urls))

	seen := make(map[string]struct{}, len(ids))
	for i, id := range ids {
		assert.Len(t, id, idLength)

		// IDs must be unique.
		_, dup := seen[id]
		assert.False(t, dup)
		seen[id] = struct{}{}

		originalURL, ok := repo.FindByID(id)
		require.True(t, ok)
		assert.Equal(t, urls[i], originalURL)
	}
}

func TestShortenerService_ShortenBatch_Empty(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := NewShortenerService(repo)

	ids, err := svc.ShortenBatch(nil)
	require.NoError(t, err)
	require.Empty(t, ids)
}
