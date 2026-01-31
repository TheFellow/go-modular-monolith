package errors

import (
	"errors"

	"github.com/urfave/cli/v3"
)

// ToCLIExit converts an error to a cli.ExitCoder with an appropriate exit code.
func ToCLIExit(err error) error {
	if err == nil {
		return nil
	}

	var exitCoder cli.ExitCoder
	if errors.As(err, &exitCoder) {
		return err
	}

	return cli.Exit(err, ExitGeneral)
}
