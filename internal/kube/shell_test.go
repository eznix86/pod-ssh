package kube

import (
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func probeInstalled(installed ...string) (func(string) error, *[]string) {
	probed := []string{}
	return func(shell string) error {
		probed = append(probed, shell)
		if slices.Contains(installed, shell) {
			return nil
		}
		return errors.New(`exec: "` + shell + `": executable file not found in $PATH`)
	}, &probed
}

func TestFindShellReturnsTheFirstInstalledCandidate(t *testing.T) {
	t.Parallel()
	probe, probed := probeInstalled("bash", "zsh")

	shell, err := findShell([]string{"sh", "bash", "zsh"}, probe)

	require.NoError(t, err)
	assert.Equal(t, "bash", shell)
	assert.Equal(t, []string{"sh", "bash"}, *probed)
}

func TestFindShellProbesEachCandidateOnce(t *testing.T) {
	t.Parallel()
	probe, probed := probeInstalled("zsh")

	shell, err := findShell([]string{"bash", "", "sh", "bash", "zsh"}, probe)

	require.NoError(t, err)
	assert.Equal(t, "zsh", shell)
	assert.Equal(t, []string{"bash", "sh", "zsh"}, *probed)
}

func TestFindShellNamesEveryShellItTriedWhenNoneIsInstalled(t *testing.T) {
	t.Parallel()
	probe, _ := probeInstalled()

	_, err := findShell([]string{"sh", "bash"}, probe)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "tried sh, bash")
}

func TestFindShellStopsOnAnErrorThatIsNotAMissingShell(t *testing.T) {
	t.Parallel()
	probed := []string{}
	unreachable := errors.New("dial tcp: connection refused")

	_, err := findShell([]string{"sh", "bash"}, func(shell string) error {
		probed = append(probed, shell)
		return unreachable
	})

	assert.ErrorIs(t, err, unreachable)
	assert.Equal(t, []string{"sh"}, probed)
}
