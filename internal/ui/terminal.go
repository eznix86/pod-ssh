package ui

import "os"

// bubbleTeaEnvironment keeps short-lived Bubble Tea programs from sending
// terminal capability queries whose replies can arrive after the program exits
// and be interpreted as input by the remote shell.
func bubbleTeaEnvironment() []string {
	environment := make([]string, 0, len(os.Environ())+2)
	hasTerm := false
	for _, entry := range os.Environ() {
		if len(entry) >= len("TERM=") && entry[:len("TERM=")] == "TERM=" {
			environment = append(environment, "TERM=xterm-256color")
			hasTerm = true
			continue
		}
		if len(entry) >= len("TERM_PROGRAM=") && entry[:len("TERM_PROGRAM=")] == "TERM_PROGRAM=" {
			continue
		}
		if len(entry) >= len("WT_SESSION=") && entry[:len("WT_SESSION=")] == "WT_SESSION=" {
			continue
		}
		if len(entry) >= len("SSH_TTY=") && entry[:len("SSH_TTY=")] == "SSH_TTY=" {
			continue
		}
		environment = append(environment, entry)
	}
	if !hasTerm {
		environment = append(environment, "TERM=xterm-256color")
	}
	environment = append(environment, "SSH_TTY=pod-ssh")
	return environment
}
