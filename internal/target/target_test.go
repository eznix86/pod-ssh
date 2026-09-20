package target

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		expected  Target
		shouldErr bool
	}{
		{name: "pod only", value: "api", expected: Target{Pod: "api"}},
		{name: "pod and namespace", value: "api@production", expected: Target{Pod: "api", Namespace: "production"}},
		{name: "empty pod", value: "@production", shouldErr: true},
		{name: "empty namespace", value: "api@", shouldErr: true},
		{name: "too many separators", value: "api@production@cluster", shouldErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual, err := Parse(test.value)
			if test.shouldErr {
				assert.ErrorIs(t, err, ErrInvalid)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}
