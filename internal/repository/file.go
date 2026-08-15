package repository

import (
	"bufio"
	"encoding/json"
	"errors"
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
func (r *FileRepository) Save(id, originalURL string) error {
	record := model.URLRecord{
		UUID:        strconv.Itoa(r.nextUUID),
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
		return err
	}
	return nil
}

// SaveBatch stores multiple URL mappings and appends them to the storage file.
func (r *FileRepository) SaveBatch(pairs []model.URLPair) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	records := make([]model.URLRecord, 0, len(pairs))
	for _, p := range pairs {
		record := model.URLRecord{
			UUID:        strconv.Itoa(r.nextUUID),
			ShortURL:    p.ShortURL,
			OriginalURL: p.OriginalURL,
		}
		records = append(records, record)
		r.urls[p.ShortURL] = p.OriginalURL
		r.nextUUID++
	}

	if err := appendRecords(r.filePath, records); err != nil {
		for _, rec := range records {
			delete(r.urls, rec.ShortURL)
			r.nextUUID--
		}
		return err
	}

	return nil
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

		uuid, err := strconv.Atoi(record.UUID)
		if err == nil && uuid >= nextUUID {
			nextUUID = uuid + 1
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}

	return urls, nextUUID, nil
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
