package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/NxthxnX/urlshortener/internal/model"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
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
// If originalURL already exists, returns the existing short URL and
// an error wrapping ErrOriginalURLConflict. If short_url collides,
// returns an error wrapping ErrShortURLConflict.
func (r *PostgresRepository) Save(id, originalURL string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var shortURL string
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO urls (short_url, original_url) VALUES ($1, $2)
		 ON CONFLICT (original_url) DO UPDATE SET original_url = EXCLUDED.original_url
		 RETURNING short_url`,
		id, originalURL,
	).Scan(&shortURL)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return "", fmt.Errorf("%w: %v", ErrShortURLConflict, err)
		}
		return "", err
	}

	if shortURL != id {
		return shortURL, fmt.Errorf("%w: original URL already exists", ErrOriginalURLConflict)
	}

	return shortURL, nil
}

// SaveBatch stores multiple URL mappings in a single transaction.
// If any original URL already exists, the existing short URL is used
// and the returned error wraps ErrOriginalURLConflict. If any short URL
// collides, the transaction is rolled back and the returned error wraps
// ErrShortURLConflict.
func (r *PostgresRepository) SaveBatch(pairs []model.URLPair) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO urls (short_url, original_url) VALUES ($1, $2)
		 ON CONFLICT (original_url) DO UPDATE SET original_url = EXCLUDED.original_url
		 RETURNING short_url`,
	)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	defer stmt.Close()

	ids := make([]string, 0, len(pairs))
	hasConflict := false

	for _, p := range pairs {
		var shortURL string
		err := stmt.QueryRowContext(ctx, p.ShortURL, p.OriginalURL).Scan(&shortURL)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				tx.Rollback()
				return nil, fmt.Errorf("%w: %v", ErrShortURLConflict, err)
			}
			tx.Rollback()
			return nil, err
		}
		if shortURL != p.ShortURL {
			hasConflict = true
		}
		ids = append(ids, shortURL)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if hasConflict {
		return ids, fmt.Errorf("%w: some original URLs already exist", ErrOriginalURLConflict)
	}

	return ids, nil
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

// Clear removes all rows from the urls table.
func (r *PostgresRepository) Clear() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `DELETE FROM urls`)
	return err
}
