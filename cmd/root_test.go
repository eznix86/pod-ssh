package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionStatusReplacesConnectingWithConnectedOnFirstOutput(t *testing.T) {
	t.Parallel()

	var terminal, output bytes.Buffer
	status := newConnectionStatus(&terminal, "api@production (app)")
	writer := status.wrap(&output)

	_, err := writer.Write([]byte("/ # "))
	require.NoError(t, err)
	_, err = writer.Write([]byte("ls\r\n"))
	require.NoError(t, err)

	assert.Equal(t, "Connecting to api@production (app)..."+clearLine+"Connected to api@production (app).\r\n", terminal.String())
	assert.Equal(t, "/ # ls\r\n", output.String())
}

func TestConnectionStatusKeepsNotesAboveTheConnectingLine(t *testing.T) {
	t.Parallel()

	var terminal bytes.Buffer
	status := newConnectionStatus(&terminal, "api@production (app)")
	status.note(`Shell "bash" is unavailable; using "sh".`)

	assert.Equal(t,
		"Connecting to api@production (app)..."+clearLine+"Shell \"bash\" is unavailable; using \"sh\".\nConnecting to api@production (app)...",
		terminal.String(),
	)
}

func TestConnectionStatusEndsTheLineWhenTheConnectionFailsBeforeOutput(t *testing.T) {
	t.Parallel()

	var terminal bytes.Buffer
	status := newConnectionStatus(&terminal, "api@production (app)")
	status.failed()
	status.connected()

	assert.Equal(t, "Connecting to api@production (app)...\n", terminal.String())
}

func TestRemoteCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		command     []string
		interactive bool
	}{
		{name: "no command opens the first installed shell", args: nil, command: nil, interactive: true},
		{name: "a known shell opens that shell", args: []string{"bash"}, command: []string{"bash"}, interactive: true},
		{name: "any other word runs as a command", args: []string{"env"}, command: []string{"env"}},
		{name: "several words run as one command", args: []string{"cat", "/var/www/nginx.conf"}, command: []string{"cat", "/var/www/nginx.conf"}},
		{name: "a shell with arguments runs as a command", args: []string{"sh", "-c", "ls -la"}, command: []string{"sh", "-c", "ls -la"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			command, interactive := remoteCommand(test.args)
			assert.Equal(t, test.command, command)
			assert.Equal(t, test.interactive, interactive)
		})
	}
}
