package repository

import (
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_UsesMemoryWhenPathEmpty(t *testing.T) {
	repo, err := New(Config{})
	require.NoError(t, err)
	_, ok := repo.(*MemoryRepository)
	assert.True(t, ok)
}

func TestNew_UsesFileWhenPathSet(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "urls.json")

	repo, err := New(Config{FileStoragePath: filePath})
	require.NoError(t, err)
	_, ok := repo.(*FileRepository)
	assert.True(t, ok)
}

func TestNew_PrefersPostgresOverFile(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	filePath := filepath.Join(t.TempDir(), "urls.json")
	repo, err := New(Config{
		DatabaseDSN:     "postgres://user:pass@localhost:5432/db",
		FileStoragePath: filePath,
		DB:              db,
	})
	require.NoError(t, err)
	_, ok := repo.(*PostgresRepository)
	assert.True(t, ok)
	require.NoError(t, mock.ExpectationsWereMet())
}
