package repository

import (
	"database/sql"

	"github.com/NxthxnX/urlshortener/internal/model"
)

// Repository defines the contract for URL persistence backends.
type Repository interface {
	Save(id, originalURL string) error
	SaveBatch(pairs []model.URLPair) error
	FindByID(id string) (string, bool)
}

// Config selects and configures a storage backend.
type Config struct {
	FileStoragePath string
	DatabaseDSN     string
	DB              *sql.DB // optional shared connection; used when DatabaseDSN is set
}

type backend struct {
	enabled func(Config) bool
	create  func(Config) (Repository, error)
}

// backends lists persistent storage backends in priority order (first match wins):
// PostgreSQL → file → in-memory fallback.
var backends = []backend{
	{
		enabled: func(cfg Config) bool {
			return cfg.DatabaseDSN != ""
		},
		create: func(cfg Config) (Repository, error) {
			if cfg.DB != nil {
				return NewPostgresRepository(cfg.DB)
			}
			return NewPostgresRepositoryFromDSN(cfg.DatabaseDSN)
		},
	},
	{
		enabled: func(cfg Config) bool {
			return cfg.FileStoragePath != ""
		},
		create: func(cfg Config) (Repository, error) {
			return NewFileRepository(cfg.FileStoragePath)
		},
	},
}

// New creates a Repository from configuration.
// Checks backends in priority order; falls back to in-memory storage.
func New(cfg Config) (Repository, error) {
	for _, b := range backends {
		if b.enabled(cfg) {
			return b.create(cfg)
		}
	}

	return NewMemoryRepository(), nil
}
