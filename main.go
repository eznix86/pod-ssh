package main

import (
	"fmt"
	"os"

	"github.com/eznix86/pod-ssh/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
