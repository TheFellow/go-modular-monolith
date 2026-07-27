package main

import (
	"bytes"
	"context"
	"fmt"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/urfave/cli/v3"
)

// cliE2E runs a newly constructed copy of the real command tree for every
// invocation. A shared database path makes persistence across process-like
// lifecycles explicit while avoiding urfave's mutable flag state.
type cliE2E struct {
	dbPath string
	actor  string
}

type cliResult struct {
	Stdout   string
	Stderr   string
	Err      error
	ExitCode int
}

func newCLIE2E(dbPath string) *cliE2E {
	return &cliE2E{dbPath: dbPath, actor: "owner"}
}

func (f *cliE2E) As(actor string) *cliE2E {
	return &cliE2E{dbPath: f.dbPath, actor: actor}
}

func (f *cliE2E) Run(args ...string) cliResult {
	c, err := NewCLI()
	if err != nil {
		return cliResult{Err: err, ExitCode: errors.ExitInternal}
	}
	c.dbPath, c.actor, c.logLevel = f.dbPath, f.actor, "error"
	cmd := c.Command()
	var stdout, stderr bytes.Buffer
	setCommandWriters(cmd, &stdout, &stderr)
	err = cmd.Run(context.Background(), append([]string{"mixology"}, args...))
	if err != nil && stderr.Len() == 0 {
		_, _ = fmt.Fprintln(&stderr, err)
	}
	result := cliResult{Stdout: stdout.String(), Stderr: stderr.String(), Err: err}
	if err != nil {
		result.ExitCode = errors.ExitGeneral
		var exit cli.ExitCoder
		if errors.As(err, &exit) {
			result.ExitCode = exit.ExitCode()
		}
	}
	return result
}

func setCommandWriters(command *cli.Command, stdout, stderr *bytes.Buffer) {
	command.Writer, command.ErrWriter = stdout, stderr
	for _, child := range command.Commands {
		setCommandWriters(child, stdout, stderr)
	}
}
