package repository

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var _ Repository = (*PostgresRepository)(nil)

// PostgresRepository stores URL mappings in PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a PostgreSQL-backed repository.
func NewPostgresRepository(db *sql.DB) (*PostgresRepository, error) {
	return &PostgresRepository{db: db}, nil
}

// NewPostgresRepositoryFromDSN opens a PostgreSQL connection from DSN
// and creates a PostgreSQL-backed repository.
func NewPostgresRepositoryFromDSN(dsn string) (*PostgresRepository, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	return NewPostgresRepository(db)
}

// Save stores a mapping between id and originalURL in PostgreSQL.
func (r *PostgresRepository) Save(id, originalURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r.db.ExecContext(ctx,
		`INSERT INTO urls (short_url, original_url) VALUES ($1, $2)
		 ON CONFLICT (short_url) DO UPDATE SET original_url = EXCLUDED.original_url`,
		id, originalURL,
	)
}

// FindByID retrieves the original URL by its shortened ID.
// Returns the original URL and a boolean indicating if found.
func (r *PostgresRepository) FindByID(id string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var originalURL string
	err := r.db.QueryRowContext(ctx,
		`SELECT original_url FROM urls WHERE short_url = $1`,
		id,
	).Scan(&originalURL)
	if err != nil {
		return "", false
	}
	return originalURL, true
}
