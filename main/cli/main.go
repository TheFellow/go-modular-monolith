package main

import (
	"context"
	"os"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/urfave/cli/v3"
)

func main() {
	cliApp, err := NewCLI()
	if err != nil {
		cli.HandleExitCoder(errors.ToCLIExit(err))
		os.Exit(errors.ExitGeneral)
	}

	cmd := cliApp.Command()
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		cli.HandleExitCoder(errors.ToCLIExit(err))
		os.Exit(errors.ExitGeneral)
	}
}
