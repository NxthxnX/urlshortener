package repository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/NxthxnX/urlshortener/internal/model"
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

	err := repo.Save("abc12345", "http://example.com")
	require.NoError(t, err)
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

	err := repo.Save("id1", "http://example.com")
	require.NoError(t, err)
	err = repo.Save("id1", "http://updated.com")
	require.NoError(t, err)
	originalURL, ok := repo.FindByID("id1")
	require.True(t, ok)
	assert.Equal(t, "http://updated.com", originalURL)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_SaveBatch(t *testing.T) {
	repo, mock := newPostgresRepoWithMock(t)

	mock.ExpectBegin()
	prep := mock.ExpectPrepare(`INSERT INTO urls`)
	prep.ExpectExec().
		WithArgs("short1", "http://yandex.ru").
		WillReturnResult(sqlmock.NewResult(1, 1))
	prep.ExpectExec().
		WithArgs("short2", "http://ya.ru").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.SaveBatch([]model.URLPair{
		{ShortURL: "short1", OriginalURL: "http://yandex.ru"},
		{ShortURL: "short2", OriginalURL: "http://ya.ru"},
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_SaveBatch_Empty(t *testing.T) {
	repo, mock := newPostgresRepoWithMock(t)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO urls`)
	mock.ExpectCommit()

	err := repo.SaveBatch(nil)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_SaveBatch_ExecError(t *testing.T) {
	repo, mock := newPostgresRepoWithMock(t)

	mock.ExpectBegin()
	prep := mock.ExpectPrepare(`INSERT INTO urls`)
	prep.ExpectExec().
		WithArgs("short1", "http://yandex.ru").
		WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()

	err := repo.SaveBatch([]model.URLPair{
		{ShortURL: "short1", OriginalURL: "http://yandex.ru"},
	})
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_SaveBatch_PrepareError(t *testing.T) {
	repo, mock := newPostgresRepoWithMock(t)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO urls`).
		WillReturnError(errors.New("prepare failed"))
	mock.ExpectRollback()

	err := repo.SaveBatch([]model.URLPair{
		{ShortURL: "short1", OriginalURL: "http://yandex.ru"},
	})
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
