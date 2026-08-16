package repository

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/NxthxnX/urlshortener/internal/model"
)

var _ Repository = (*FileRepository)(nil)

// FileRepository stores URL mappings in memory and persists them to a JSON-lines file.
type FileRepository struct {
	mu          sync.RWMutex
	filePath    string
	urls        map[string]string // shortURL -> originalURL
	origToShort map[string]string // originalURL -> shortURL
	nextUUID    int
}

// NewFileRepository creates a file-backed repository and loads existing records from disk.
func NewFileRepository(filePath string) (*FileRepository, error) {
	filePath = filepath.Clean(filePath)

	if dir := filepath.Dir(filePath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	urls, origToShort, nextUUID, err := loadFromFile(filePath)
	if err != nil {
		return nil, err
	}

	return &FileRepository{
		filePath:    filePath,
		urls:        urls,
		origToShort: origToShort,
		nextUUID:    nextUUID,
	}, nil
}

// Save stores a mapping between id and originalURL and appends it to the storage file.
// If originalURL already exists, returns the existing short URL and
// an error wrapping ErrOriginalURLConflict. If short_url collides,
// returns an error wrapping ErrShortURLConflict.
func (r *FileRepository) Save(id, originalURL string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existingID, ok := r.origToShort[originalURL]; ok {
		return existingID, fmt.Errorf("%w: original URL already exists", ErrOriginalURLConflict)
	}

	if _, exists := r.urls[id]; exists {
		return "", fmt.Errorf("%w: short URL already exists", ErrShortURLConflict)
	}

	record := model.URLRecord{
		UUID:        strconv.Itoa(r.nextUUID),
		ShortURL:    id,
		OriginalURL: originalURL,
	}

	r.urls[id] = originalURL
	r.origToShort[originalURL] = id
	r.nextUUID++

	if err := appendRecord(r.filePath, record); err != nil {
		delete(r.urls, id)
		delete(r.origToShort, originalURL)
		r.nextUUID--
		return "", err
	}
	return id, nil
}

// SaveBatch stores multiple URL mappings and appends them to the storage file.
// If any original URL already exists, the existing short URL is used
// and the returned error wraps ErrOriginalURLConflict.
func (r *FileRepository) SaveBatch(pairs []model.URLPair) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ids := make([]string, 0, len(pairs))
	records := make([]model.URLRecord, 0, len(pairs))
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

		record := model.URLRecord{
			UUID:        strconv.Itoa(r.nextUUID),
			ShortURL:    p.ShortURL,
			OriginalURL: p.OriginalURL,
		}
		records = append(records, record)
		r.urls[p.ShortURL] = p.OriginalURL
		r.origToShort[p.OriginalURL] = p.ShortURL
		r.nextUUID++
		ids = append(ids, p.ShortURL)
	}

	if err := appendRecords(r.filePath, records); err != nil {
		for _, rec := range records {
			delete(r.urls, rec.ShortURL)
			delete(r.origToShort, rec.OriginalURL)
			r.nextUUID--
		}
		return nil, err
	}

	if hasConflict {
		return ids, fmt.Errorf("%w: some original URLs already exist", ErrOriginalURLConflict)
	}

	return ids, nil
}

// FindByID retrieves the original URL by its shortened ID.
// Returns the original URL and a boolean indicating if found.
func (r *FileRepository) FindByID(id string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	originalURL, ok := r.urls[id]
	return originalURL, ok
}

// Clear removes all stored URL mappings and truncates the storage file.
func (r *FileRepository) Clear() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.urls = make(map[string]string)
	r.origToShort = make(map[string]string)
	r.nextUUID = 1

	if err := os.Truncate(r.filePath, 0); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func loadFromFile(filePath string) (map[string]string, map[string]string, int, error) {
	urls := make(map[string]string)
	origToShort := make(map[string]string)
	nextUUID := 1

	file, err := os.Open(filePath)
	if errors.Is(err, fs.ErrNotExist) {
		return urls, origToShort, nextUUID, nil
	}
	if err != nil {
		return nil, nil, 0, err
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
		origToShort[record.OriginalURL] = record.ShortURL

		uuid, err := strconv.Atoi(record.UUID)
		if err == nil && uuid >= nextUUID {
			nextUUID = uuid + 1
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, 0, err
	}

	return urls, origToShort, nextUUID, nil
}

func appendRecord(filePath string, record model.URLRecord) error {
	return appendRecords(filePath, []model.URLRecord{record})
}

// appendRecords writes the given records as JSON-lines to the storage file
// in a single batch (single file open).
func appendRecords(filePath string, records []model.URLRecord) error {
	if len(records) == 0 {
		return nil
	}

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, record := range records {
		data, err := json.Marshal(record)
		if err != nil {
			return err
		}
		data = append(data, '\n')

		if _, err := writer.Write(data); err != nil {
			return err
		}
	}

	return writer.Flush()
}
