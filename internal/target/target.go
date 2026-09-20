// Package target parses pod-ssh connection targets.
package target

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalid is returned when a target is malformed.
var ErrInvalid = errors.New("invalid target")

// Target identifies a pod and an optional namespace.
type Target struct {
	Pod       string
	Namespace string
}

// Parse parses pod and pod@namespace target forms.
func Parse(value string) (Target, error) {
	parts := strings.Split(value, "@")
	if len(parts) > 2 || strings.TrimSpace(parts[0]) == "" {
		return Target{}, fmt.Errorf("%w %q: expected pod or pod@namespace", ErrInvalid, value)
	}

	parsed := Target{Pod: strings.TrimSpace(parts[0])}
	if len(parts) == 2 {
		parsed.Namespace = strings.TrimSpace(parts[1])
		if parsed.Namespace == "" {
			return Target{}, fmt.Errorf("%w %q: namespace is empty", ErrInvalid, value)
		}
	}
	return parsed, nil
}

// String returns the canonical pod@namespace representation.
func (t Target) String() string {
	if t.Namespace == "" {
		return t.Pod
	}
	return t.Pod + "@" + t.Namespace
}
