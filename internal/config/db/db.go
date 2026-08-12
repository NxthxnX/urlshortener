package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/NxthxnX/urlshortener/migrations"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
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

// DB returns the underlying *sql.DB, or nil if not configured.
func (s *Store) DB() *sql.DB {
	if s == nil {
		return nil
	}

	return s.db
}

// Close closes the underlying database connection.
func (s *Store) Close() {
	if s != nil && s.db != nil {
		s.db.Close()
	}
}

// Migrate applies SQL migrations from the embedded migrations package.
func (s *Store) Migrate() error {
	if s == nil || s.db == nil {
		return nil
	}

	dbDriver, err := postgres.WithInstance(s.db, &postgres.Config{})
	if err != nil {
		return err
	}

	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		sourceDriver,
		"postgres",
		dbDriver,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
