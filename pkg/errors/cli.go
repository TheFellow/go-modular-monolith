package errors

import (
	"github.com/urfave/cli/v3"
)

// ToCLIExit converts an error to a cli.ExitCoder with an appropriate exit code.
func ToCLIExit(err error) error {
	if err == nil {
		return nil
	}

	if _, ok := err.(cli.ExitCoder); ok {
		return err
	}

	return cli.Exit(err, ExitGeneral)
}
