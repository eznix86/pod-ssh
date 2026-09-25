package cmd

import (
	"fmt"
	"io"
	"sync"

	"github.com/fatih/color"
)

const clearLine = "\r\033[2K"

type connectionStatus struct {
	terminal io.Writer
	target   string
	mu       sync.Mutex
	settled  bool
}

func newConnectionStatus(terminal io.Writer, target string) *connectionStatus {
	status := &connectionStatus{terminal: terminal, target: target}
	status.printConnecting()
	return status
}

func (s *connectionStatus) printConnecting() {
	color.New(color.FgCyan).Fprintf(s.terminal, "Connecting to %s...", s.target)
}

func (s *connectionStatus) connected() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settled {
		return
	}
	s.settled = true
	fmt.Fprint(s.terminal, clearLine)
	color.New(color.FgGreen).Fprintf(s.terminal, "Connected to %s.", s.target)
	fmt.Fprint(s.terminal, "\r\n")
}

func (s *connectionStatus) note(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settled {
		return
	}
	fmt.Fprint(s.terminal, clearLine+message+"\n")
	s.printConnecting()
}

func (s *connectionStatus) failed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settled {
		return
	}
	s.settled = true
	fmt.Fprintln(s.terminal)
}

func (s *connectionStatus) wrap(writer io.Writer) io.Writer {
	return statusWriter{status: s, writer: writer}
}

type statusWriter struct {
	status *connectionStatus
	writer io.Writer
}

func (w statusWriter) Write(data []byte) (int, error) {
	w.status.connected()
	return w.writer.Write(data)
}
