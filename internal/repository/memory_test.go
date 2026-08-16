package repository

import (
	"errors"
	"testing"

	"github.com/NxthxnX/urlshortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryRepository_SaveAndFind(t *testing.T) {
	repo := NewMemoryRepository()

	shortURL, err := repo.Save("abc12345", "http://example.com")
	require.NoError(t, err)
	assert.Equal(t, "abc12345", shortURL)

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

	shortURL, err := repo.Save("id1", "http://example.com")
	require.NoError(t, err)
	assert.Equal(t, "id1", shortURL)

	_, err = repo.Save("id1", "http://updated.com")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrShortURLConflict))

	originalURL, ok := repo.FindByID("id1")
	require.True(t, ok)
	assert.Equal(t, "http://example.com", originalURL)
}

func TestMemoryRepository_MultipleEntries(t *testing.T) {
	repo := NewMemoryRepository()

	shortURL, err := repo.Save("short1", "http://yandex.ru")
	require.NoError(t, err)
	assert.Equal(t, "short1", shortURL)
	shortURL, err = repo.Save("short2", "http://ya.ru")
	require.NoError(t, err)
	assert.Equal(t, "short2", shortURL)

	originalURL, ok := repo.FindByID("short1")
	require.True(t, ok)
	assert.Equal(t, "http://yandex.ru", originalURL)

	originalURL, ok = repo.FindByID("short2")
	require.True(t, ok)
	assert.Equal(t, "http://ya.ru", originalURL)
}

func TestMemoryRepository_SaveBatch(t *testing.T) {
	repo := NewMemoryRepository()

	ids, err := repo.SaveBatch([]model.URLPair{
		{ShortURL: "short1", OriginalURL: "http://yandex.ru"},
		{ShortURL: "short2", OriginalURL: "http://ya.ru"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"short1", "short2"}, ids)

	originalURL, ok := repo.FindByID("short1")
	require.True(t, ok)
	assert.Equal(t, "http://yandex.ru", originalURL)

	originalURL, ok = repo.FindByID("short2")
	require.True(t, ok)
	assert.Equal(t, "http://ya.ru", originalURL)

	// empty batch is a no-op
	emptyIDs, err := repo.SaveBatch(nil)
	require.NoError(t, err)
	assert.Empty(t, emptyIDs)
}

func TestMemoryRepository_Save_OriginalURLConflict(t *testing.T) {
	repo := NewMemoryRepository()

	shortURL, err := repo.Save("id1", "http://example.com")
	require.NoError(t, err)
	assert.Equal(t, "id1", shortURL)

	existingShortURL, err := repo.Save("id2", "http://example.com")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrOriginalURLConflict))
	assert.Equal(t, "id1", existingShortURL)
}

func TestMemoryRepository_Save_ShortURLConflict(t *testing.T) {
	repo := NewMemoryRepository()

	shortURL, err := repo.Save("id1", "http://example.com")
	require.NoError(t, err)
	assert.Equal(t, "id1", shortURL)

	_, err = repo.Save("id1", "http://other.com")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrShortURLConflict))
}

func TestMemoryRepository_SaveBatch_OriginalURLConflict(t *testing.T) {
	repo := NewMemoryRepository()

	_, err := repo.Save("existing", "http://yandex.ru")
	require.NoError(t, err)

	ids, err := repo.SaveBatch([]model.URLPair{
		{ShortURL: "new1", OriginalURL: "http://new.com"},
		{ShortURL: "new2", OriginalURL: "http://yandex.ru"},
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrOriginalURLConflict))
	assert.Equal(t, []string{"new1", "existing"}, ids)
}

func TestMemoryRepository_Clear(t *testing.T) {
	repo := NewMemoryRepository()

	_, err := repo.Save("short1", "http://yandex.ru")
	require.NoError(t, err)
	_, err = repo.Save("short2", "http://ya.ru")
	require.NoError(t, err)

	require.NoError(t, repo.Clear())

	_, ok := repo.FindByID("short1")
	assert.False(t, ok)
	_, ok = repo.FindByID("short2")
	assert.False(t, ok)

	shortURL, err := repo.Save("short1", "http://yandex.ru")
	require.NoError(t, err)
	assert.Equal(t, "short1", shortURL)
}
