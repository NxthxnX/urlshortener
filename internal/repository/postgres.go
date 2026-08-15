package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/NxthxnX/urlshortener/internal/model"
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
func (r *PostgresRepository) Save(id, originalURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO urls (short_url, original_url) VALUES ($1, $2)
		 ON CONFLICT (short_url) DO UPDATE SET original_url = EXCLUDED.original_url`,
		id, originalURL,
	)
	return err
}

// SaveBatch stores multiple URL mappings in a single transaction.
func (r *PostgresRepository) SaveBatch(pairs []model.URLPair) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO urls (short_url, original_url) VALUES ($1, $2)
		 ON CONFLICT (short_url) DO UPDATE SET original_url = EXCLUDED.original_url`,
	)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, p := range pairs {
		if _, err := stmt.ExecContext(ctx, p.ShortURL, p.OriginalURL); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
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
