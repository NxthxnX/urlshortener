package service

import (
	"errors"
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

func TestShortenerService_Shorten_Conflict(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := NewShortenerService(repo)

	id1, err := svc.Shorten("http://example.com")
	require.NoError(t, err)
	assert.Len(t, id1, idLength)

	id2, err := svc.Shorten("http://example.com")
	require.Error(t, err)
	assert.True(t, errors.Is(err, repository.ErrOriginalURLConflict))
	assert.Equal(t, id1, id2)
}

func TestShortenerService_ShortenBatch_Conflict(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := NewShortenerService(repo)

	existingID, err := svc.Shorten("http://yandex.ru")
	require.NoError(t, err)

	ids, err := svc.ShortenBatch([]string{"http://new.com", "http://yandex.ru"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, repository.ErrOriginalURLConflict))
	require.Len(t, ids, 2)

	assert.Len(t, ids[0], idLength)
	assert.Equal(t, existingID, ids[1])
}
