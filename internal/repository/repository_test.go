package repository

import (
	"path/filepath"
	"testing"

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
