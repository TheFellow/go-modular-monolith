package main

import (
	"context"
	"os"

	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/urfave/cli/v3"
)

func main() {
	cliApp, err := NewCLI()
	if err != nil {
		cli.HandleExitCoder(apperrors.ToCLIExit(err))
		os.Exit(apperrors.ExitGeneral)
	}

	cmd := cliApp.Command()
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		cli.HandleExitCoder(apperrors.ToCLIExit(err))
		os.Exit(apperrors.ExitGeneral)
	}
}
