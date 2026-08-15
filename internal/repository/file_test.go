package repository

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/NxthxnX/urlshortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readRecordsFromFile(t *testing.T, filePath string) []model.URLRecord {
	t.Helper()

	file, err := os.Open(filePath)
	require.NoError(t, err)
	defer file.Close()

	var records []model.URLRecord
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var record model.URLRecord
		require.NoError(t, json.Unmarshal(line, &record))
		records = append(records, record)
	}
	require.NoError(t, scanner.Err())

	return records
}

func TestFileRepository_SaveAndFind(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "urls.json")

	repo, err := NewFileRepository(filePath)
	require.NoError(t, err)

	err = repo.Save("abc12345", "http://example.com")
	require.NoError(t, err)

	originalURL, ok := repo.FindByID("abc12345")
	require.True(t, ok)
	assert.Equal(t, "http://example.com", originalURL)

	records := readRecordsFromFile(t, filePath)
	require.Len(t, records, 1)
	assert.Equal(t, "1", records[0].UUID)
	assert.Equal(t, "abc12345", records[0].ShortURL)
	assert.Equal(t, "http://example.com", records[0].OriginalURL)

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.NotContains(t, string(content), "[")
}

func TestFileRepository_RestoreOnRestart(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "urls.json")

	repo, err := NewFileRepository(filePath)
	require.NoError(t, err)

	err = repo.Save("short1", "http://yandex.ru")
	require.NoError(t, err)
	err = repo.Save("short2", "http://ya.ru")
	require.NoError(t, err)

	restartedRepo, err := NewFileRepository(filePath)
	require.NoError(t, err)

	records := readRecordsFromFile(t, filePath)
	require.Len(t, records, 2)

	originalURL, ok := restartedRepo.FindByID("short1")
	require.True(t, ok)
	assert.Equal(t, "http://yandex.ru", originalURL)

	originalURL, ok = restartedRepo.FindByID("short2")
	require.True(t, ok)
	assert.Equal(t, "http://ya.ru", originalURL)
}

func TestNewFileRepository_CreatesParentDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tmp")
	filePath := filepath.Join(dir, "short-url-db.json")

	repo, err := NewFileRepository(filePath)
	require.NoError(t, err)
	require.DirExists(t, dir)

	err = repo.Save("abc12345", "http://example.com")
	require.NoError(t, err)
	require.FileExists(t, filePath)
}

func TestFileRepository_SaveBatch(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "urls.json")

	repo, err := NewFileRepository(filePath)
	require.NoError(t, err)

	err = repo.SaveBatch([]model.URLPair{
		{ShortURL: "short1", OriginalURL: "http://yandex.ru"},
		{ShortURL: "short2", OriginalURL: "http://ya.ru"},
	})
	require.NoError(t, err)

	records := readRecordsFromFile(t, filePath)
	require.Len(t, records, 2)
	assert.Equal(t, "1", records[0].UUID)
	assert.Equal(t, "short1", records[0].ShortURL)
	assert.Equal(t, "http://yandex.ru", records[0].OriginalURL)
	assert.Equal(t, "2", records[1].UUID)
	assert.Equal(t, "short2", records[1].ShortURL)
	assert.Equal(t, "http://ya.ru", records[1].OriginalURL)

	originalURL, ok := repo.FindByID("short1")
	require.True(t, ok)
	assert.Equal(t, "http://yandex.ru", originalURL)

	originalURL, ok = repo.FindByID("short2")
	require.True(t, ok)
	assert.Equal(t, "http://ya.ru", originalURL)
}

func TestFileRepository_SaveBatch_RestoreOnRestart(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "urls.json")

	repo, err := NewFileRepository(filePath)
	require.NoError(t, err)

	require.NoError(t, repo.SaveBatch([]model.URLPair{
		{ShortURL: "short1", OriginalURL: "http://yandex.ru"},
		{ShortURL: "short2", OriginalURL: "http://ya.ru"},
	}))

	err = repo.Save("short3", "http://go.dev")
	require.NoError(t, err)

	restartedRepo, err := NewFileRepository(filePath)
	require.NoError(t, err)

	originalURL, ok := restartedRepo.FindByID("short1")
	require.True(t, ok)
	assert.Equal(t, "http://yandex.ru", originalURL)

	originalURL, ok = restartedRepo.FindByID("short2")
	require.True(t, ok)
	assert.Equal(t, "http://ya.ru", originalURL)

	originalURL, ok = restartedRepo.FindByID("short3")
	require.True(t, ok)
	assert.Equal(t, "http://go.dev", originalURL)

	records := readRecordsFromFile(t, filePath)
	require.Len(t, records, 3)
	assert.Equal(t, "3", records[2].UUID)
	assert.Equal(t, "short3", records[2].ShortURL)
}
