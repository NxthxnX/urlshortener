package repository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPostgresRepoWithMock(t *testing.T) (*PostgresRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	repo, err := NewPostgresRepository(db)
	require.NoError(t, err)
	return repo, mock
}

func TestPostgresRepository_SaveAndFind(t *testing.T) {
	repo, mock := newPostgresRepoWithMock(t)

	mock.ExpectExec(`INSERT INTO urls`).
		WithArgs("abc12345", "http://example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(`SELECT original_url FROM urls WHERE short_url = \$1`).
		WithArgs("abc12345").
		WillReturnRows(sqlmock.NewRows([]string{"original_url"}).AddRow("http://example.com"))

	repo.Save("abc12345", "http://example.com")
	originalURL, ok := repo.FindByID("abc12345")
	require.True(t, ok)
	assert.Equal(t, "http://example.com", originalURL)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_FindByID_NotFound(t *testing.T) {
	repo, mock := newPostgresRepoWithMock(t)

	mock.ExpectQuery(`SELECT original_url FROM urls WHERE short_url = \$1`).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows([]string{"original_url"}))

	_, ok := repo.FindByID("missing")
	assert.False(t, ok)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_Overwrite(t *testing.T) {
	repo, mock := newPostgresRepoWithMock(t)

	mock.ExpectExec(`INSERT INTO urls`).
		WithArgs("id1", "http://example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`INSERT INTO urls`).
		WithArgs("id1", "http://updated.com").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(`SELECT original_url FROM urls WHERE short_url = \$1`).
		WithArgs("id1").
		WillReturnRows(sqlmock.NewRows([]string{"original_url"}).AddRow("http://updated.com"))

	repo.Save("id1", "http://example.com")
	repo.Save("id1", "http://updated.com")
	originalURL, ok := repo.FindByID("id1")
	require.True(t, ok)
	assert.Equal(t, "http://updated.com", originalURL)
	require.NoError(t, mock.ExpectationsWereMet())
}
