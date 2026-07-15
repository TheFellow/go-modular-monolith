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

	var appErr *Error
	if errors.As(err, &appErr) {
		return cli.Exit(appErr.UserMessage(), appErr.CLIExitCode())
	}

	if _, ok := errors.AsType[cli.ExitCoder](err); ok {
		return err
	}

	return cli.Exit(err, ExitGeneral)
}
