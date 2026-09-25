package kube

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
)

// Shells lists the shells pod-ssh knows how to start, in order of preference.
var Shells = []string{"sh", "bash", "zsh"}

// FindShell returns the first of candidates that is installed in the container.
func (c *Client) FindShell(ctx context.Context, namespace, pod, container string, candidates ...string) (string, error) {
	return findShell(candidates, func(shell string) error {
		return c.Exec(ctx, ExecOptions{
			Namespace: namespace,
			Pod:       pod,
			Container: container,
			Command:   []string{shell, "-c", "exit 0"},
			Stdout:    io.Discard,
			Stderr:    io.Discard,
		})
	})
}

func findShell(candidates []string, probe func(shell string) error) (string, error) {
	tried := make([]string, 0, len(candidates))
	var lastErr error
	for _, shell := range candidates {
		if shell == "" || slices.Contains(tried, shell) {
			continue
		}
		tried = append(tried, shell)
		err := probe(shell)
		if err == nil {
			return shell, nil
		}
		if !IsCommandNotFound(err) {
			return "", err
		}
		lastErr = err
	}
	return "", fmt.Errorf("no supported shell found in container (tried %s): %w", strings.Join(tried, ", "), lastErr)
}
