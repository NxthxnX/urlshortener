package db

import (
	"context"
	"database/sql"
	"errors"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var ErrNotConfigured = errors.New("database is not configured")

// Store wraps a PostgreSQL connection via database/sql and the pgx driver.
type Store struct {
	db *sql.DB
}

// Open creates a database connection for the given DSN.
// An empty DSN returns a Store without an active connection.
func Open(dsn string) (*Store, error) {
	if dsn == "" {
		return &Store{}, nil
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

// Ping checks the database connection.
func (s *Store) Ping(ctx context.Context) error {
	if s == nil || s.db == nil {
		return ErrNotConfigured
	}
	return s.db.PingContext(ctx)
}

// Close closes the underlying database connection.
func (s *Store) Close() {
	if s != nil && s.db != nil {
		s.db.Close()
	}
}
