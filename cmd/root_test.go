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
