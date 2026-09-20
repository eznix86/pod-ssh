// Package history persists connection history compatible with the Python version.
package history

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const lastLimit = 20

// ErrNotFound is returned when a requested history entry does not exist.
var ErrNotFound = errors.New("history entry not found")

// Store persists unique history and recent connections.
type Store struct {
	historyPath string
	lastPath    string
}

// New creates a store using the legacy pod-ssh files in the user's home directory.
func New() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	return &Store{
		historyPath: filepath.Join(home, ".pod_ssh_history"),
		lastPath:    filepath.Join(home, ".pod_ssh_last"),
	}, nil
}

// List returns unique connection history.
func (s *Store) List() ([]string, error) {
	return readLines(s.historyPath)
}

// Recent returns connections ordered from newest to oldest.
func (s *Store) Recent() ([]string, error) {
	return readLines(s.lastPath)
}

// Save records a successful connection.
func (s *Store) Save(entry string) error {
	historyEntries, err := s.List()
	if err != nil {
		return err
	}
	found := slices.Contains(historyEntries, entry)
	if !found {
		historyEntries = append(historyEntries, entry)
	}
	if err := writeLines(s.historyPath, historyEntries); err != nil {
		return fmt.Errorf("save history: %w", err)
	}

	recentEntries, err := s.Recent()
	if err != nil {
		return err
	}
	recentEntries = append([]string{entry}, recentEntries...)
	if len(recentEntries) > lastLimit {
		recentEntries = recentEntries[:lastLimit]
	}
	if err := writeLines(s.lastPath, recentEntries); err != nil {
		return fmt.Errorf("save recent connections: %w", err)
	}
	return nil
}

// At returns the one-based recent connection at position.
func (s *Store) At(position int) (string, error) {
	entries, err := s.Recent()
	if err != nil {
		return "", err
	}
	if position < 1 || position > len(entries) {
		return "", fmt.Errorf("%w at position %d", ErrNotFound, position)
	}
	return entries[position-1], nil
}

// Clear removes the unique history file.
func (s *Store) Clear() error {
	err := os.Remove(s.historyPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear history: %w", err)
	}
	return nil
}

func readLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	entries := make([]string, 0)
	for line := range strings.SplitSeq(string(data), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			entries = append(entries, trimmed)
		}
	}
	return entries, nil
}

func writeLines(path string, entries []string) error {
	content := strings.Join(entries, "\n")
	if content != "" {
		content += "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
