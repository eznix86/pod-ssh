package history

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreSaveAndRead(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	store := &Store{
		historyPath: filepath.Join(directory, "history"),
		lastPath:    filepath.Join(directory, "last"),
	}

	require.NoError(t, store.Save("api@production"))
	require.NoError(t, store.Save("worker@production"))
	require.NoError(t, store.Save("api@production"))

	historyEntries, err := store.List()
	require.NoError(t, err)
	assert.Equal(t, []string{"api@production", "worker@production"}, historyEntries)

	recentEntries, err := store.Recent()
	require.NoError(t, err)
	assert.Equal(t, []string{"api@production", "worker@production", "api@production"}, recentEntries)

	entry, err := store.At(2)
	require.NoError(t, err)
	assert.Equal(t, "worker@production", entry)
}

func TestStoreAtRejectsInvalidPosition(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	store := &Store{
		historyPath: filepath.Join(directory, "history"),
		lastPath:    filepath.Join(directory, "last"),
	}

	_, err := store.At(1)
	assert.ErrorIs(t, err, ErrNotFound)
}
