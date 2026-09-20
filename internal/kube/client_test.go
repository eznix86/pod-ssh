package kube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchingPods(t *testing.T) {
	t.Parallel()
	pods := []Pod{{Name: "api-123"}, {Name: "api-worker-456"}, {Name: "web-789"}}

	tests := []struct {
		name     string
		query    string
		expected []Pod
	}{
		{name: "exact wins", query: "api-123", expected: []Pod{{Name: "api-123"}}},
		{name: "partial", query: "api", expected: []Pod{{Name: "api-123"}, {Name: "api-worker-456"}}},
		{name: "none", query: "missing", expected: []Pod{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.expected, MatchingPods(pods, test.query))
		})
	}
}
