package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryRepository_SaveAndFind(t *testing.T) {
	repo := NewMemoryRepository()

	repo.Save("abc12345", "http://example.com")

	originalURL, ok := repo.FindByID("abc12345")
	require.True(t, ok)
	assert.Equal(t, "http://example.com", originalURL)
}

func TestMemoryRepository_FindByID_NotFound(t *testing.T) {
	repo := NewMemoryRepository()

	_, ok := repo.FindByID("missing")
	assert.False(t, ok)
}

func TestMemoryRepository_Overwrite(t *testing.T) {
	repo := NewMemoryRepository()

	repo.Save("id1", "http://example.com")
	repo.Save("id1", "http://updated.com")

	originalURL, ok := repo.FindByID("id1")
	require.True(t, ok)
	assert.Equal(t, "http://updated.com", originalURL)
}

func TestMemoryRepository_MultipleEntries(t *testing.T) {
	repo := NewMemoryRepository()

	repo.Save("short1", "http://yandex.ru")
	repo.Save("short2", "http://ya.ru")

	originalURL, ok := repo.FindByID("short1")
	require.True(t, ok)
	assert.Equal(t, "http://yandex.ru", originalURL)

	originalURL, ok = repo.FindByID("short2")
	require.True(t, ok)
	assert.Equal(t, "http://ya.ru", originalURL)
}
