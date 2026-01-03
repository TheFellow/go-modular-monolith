package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	cliApp, err := NewCLI()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmd := cliApp.Command()
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		_, _ = fmt.Fprintln(cmd.ErrWriter, err)
		os.Exit(1)
	}
}
