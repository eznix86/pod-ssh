package main

import (
	"errors"
	"fmt"
	"os"

	utilexec "k8s.io/client-go/util/exec"

	"github.com/eznix86/pod-ssh/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		if exitErr, ok := errors.AsType[utilexec.ExitError](err); ok {
			os.Exit(exitErr.ExitStatus())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
