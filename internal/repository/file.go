package repository

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/NxthxnX/urlshortener/internal/model"
)

var _ Repository = (*FileRepository)(nil)

// FileRepository stores URL mappings in memory and persists them to a JSON-lines file.
type FileRepository struct {
	mu       sync.RWMutex
	filePath string
	urls     map[string]string // id -> originalURL
	nextUUID int
}

// NewFileRepository creates a file-backed repository and loads existing records from disk.
func NewFileRepository(filePath string) (*FileRepository, error) {
	filePath = filepath.Clean(filePath)

	if dir := filepath.Dir(filePath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	urls, nextUUID, err := loadFromFile(filePath)
	if err != nil {
		return nil, err
	}

	return &FileRepository{
		filePath: filePath,
		urls:     urls,
		nextUUID: nextUUID,
	}, nil
}

// Save stores a mapping between id and originalURL and appends it to the storage file.
func (r *FileRepository) Save(id, originalURL string) {
	record := model.URLRecord{
		UUID:        r.nextUUID,
		ShortURL:    id,
		OriginalURL: originalURL,
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.urls[id] = originalURL
	r.nextUUID++

	if err := appendRecord(r.filePath, record); err != nil {
		delete(r.urls, id)
		r.nextUUID--
	}
}

// FindByID retrieves the original URL by its shortened ID.
// Returns the original URL and a boolean indicating if found.
func (r *FileRepository) FindByID(id string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	originalURL, ok := r.urls[id]
	return originalURL, ok
}

func loadFromFile(filePath string) (map[string]string, int, error) {
	urls := make(map[string]string)
	nextUUID := 1

	file, err := os.Open(filePath)
	if errors.Is(err, fs.ErrNotExist) {
		return urls, nextUUID, nil
	}
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var record model.URLRecord
		if err := json.Unmarshal(line, &record); err != nil {
			continue
		}

		urls[record.ShortURL] = record.OriginalURL

		if record.UUID >= nextUUID {
			nextUUID = record.UUID + 1
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}

	return urls, nextUUID, nil
}

func appendRecord(filePath string, record model.URLRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}
