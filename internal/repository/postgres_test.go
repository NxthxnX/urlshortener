package repository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/NxthxnX/urlshortener/internal/model"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
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

	mock.ExpectQuery(`INSERT INTO urls`).
		WithArgs("abc12345", "http://example.com").
		WillReturnRows(sqlmock.NewRows([]string{"short_url"}).AddRow("abc12345"))

	mock.ExpectQuery(`SELECT original_url FROM urls WHERE short_url = \$1`).
		WithArgs("abc12345").
		WillReturnRows(sqlmock.NewRows([]string{"original_url"}).AddRow("http://example.com"))

	shortURL, err := repo.Save("abc12345", "http://example.com")
	require.NoError(t, err)
	assert.Equal(t, "abc12345", shortURL)
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

	mock.ExpectQuery(`INSERT INTO urls`).
		WithArgs("id1", "http://example.com").
		WillReturnRows(sqlmock.NewRows([]string{"short_url"}).AddRow("id1"))

	mock.ExpectQuery(`INSERT INTO urls`).
		WithArgs("id1", "http://updated.com").
		WillReturnError(&pgconn.PgError{Code: pgerrcode.UniqueViolation})

	mock.ExpectQuery(`SELECT original_url FROM urls WHERE short_url = \$1`).
		WithArgs("id1").
		WillReturnRows(sqlmock.NewRows([]string{"original_url"}).AddRow("http://example.com"))

	shortURL, err := repo.Save("id1", "http://example.com")
	require.NoError(t, err)
	assert.Equal(t, "id1", shortURL)
	_, err = repo.Save("id1", "http://updated.com")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrShortURLConflict))
	originalURL, ok := repo.FindByID("id1")
	require.True(t, ok)
	assert.Equal(t, "http://example.com", originalURL)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_SaveBatch(t *testing.T) {
	repo, mock := newPostgresRepoWithMock(t)

	mock.ExpectBegin()
	prep := mock.ExpectPrepare(`INSERT INTO urls`)
	prep.ExpectQuery().
		WithArgs("short1", "http://yandex.ru").
		WillReturnRows(sqlmock.NewRows([]string{"short_url"}).AddRow("short1"))
	prep.ExpectQuery().
		WithArgs("short2", "http://ya.ru").
		WillReturnRows(sqlmock.NewRows([]string{"short_url"}).AddRow("short2"))
	mock.ExpectCommit()

	ids, err := repo.SaveBatch([]model.URLPair{
		{ShortURL: "short1", OriginalURL: "http://yandex.ru"},
		{ShortURL: "short2", OriginalURL: "http://ya.ru"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"short1", "short2"}, ids)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_SaveBatch_Empty(t *testing.T) {
	repo, mock := newPostgresRepoWithMock(t)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO urls`)
	mock.ExpectCommit()

	ids, err := repo.SaveBatch(nil)
	require.NoError(t, err)
	assert.Empty(t, ids)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_SaveBatch_ExecError(t *testing.T) {
	repo, mock := newPostgresRepoWithMock(t)

	mock.ExpectBegin()
	prep := mock.ExpectPrepare(`INSERT INTO urls`)
	prep.ExpectQuery().
		WithArgs("short1", "http://yandex.ru").
		WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()

	ids, err := repo.SaveBatch([]model.URLPair{
		{ShortURL: "short1", OriginalURL: "http://yandex.ru"},
	})
	require.Error(t, err)
	assert.Nil(t, ids)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_SaveBatch_PrepareError(t *testing.T) {
	repo, mock := newPostgresRepoWithMock(t)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO urls`).
		WillReturnError(errors.New("prepare failed"))
	mock.ExpectRollback()

	ids, err := repo.SaveBatch([]model.URLPair{
		{ShortURL: "short1", OriginalURL: "http://yandex.ru"},
	})
	require.Error(t, err)
	assert.Nil(t, ids)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_Save_OriginalURLConflict(t *testing.T) {
	repo, mock := newPostgresRepoWithMock(t)

	mock.ExpectQuery(`INSERT INTO urls`).
		WithArgs("newid", "http://example.com").
		WillReturnRows(sqlmock.NewRows([]string{"short_url"}).AddRow("existingid"))

	shortURL, err := repo.Save("newid", "http://example.com")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrOriginalURLConflict))
	assert.Equal(t, "existingid", shortURL)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_SaveBatch_OriginalURLConflict(t *testing.T) {
	repo, mock := newPostgresRepoWithMock(t)

	mock.ExpectBegin()
	prep := mock.ExpectPrepare(`INSERT INTO urls`)
	prep.ExpectQuery().
		WithArgs("short1", "http://yandex.ru").
		WillReturnRows(sqlmock.NewRows([]string{"short_url"}).AddRow("short1"))
	prep.ExpectQuery().
		WithArgs("short2", "http://ya.ru").
		WillReturnRows(sqlmock.NewRows([]string{"short_url"}).AddRow("existingid"))
	mock.ExpectCommit()

	ids, err := repo.SaveBatch([]model.URLPair{
		{ShortURL: "short1", OriginalURL: "http://yandex.ru"},
		{ShortURL: "short2", OriginalURL: "http://ya.ru"},
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrOriginalURLConflict))
	assert.Equal(t, []string{"short1", "existingid"}, ids)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_Clear(t *testing.T) {
	repo, mock := newPostgresRepoWithMock(t)

	mock.ExpectExec(`DELETE FROM urls`).
		WillReturnResult(sqlmock.NewResult(0, 3))

	require.NoError(t, repo.Clear())
	require.NoError(t, mock.ExpectationsWereMet())
}
