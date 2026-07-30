package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestStandaloneTUICommandProvidesHelpWithoutBootstrapping(t *testing.T) {
	t.Parallel()

	command := newCommand()
	var output bytes.Buffer
	command.Writer = &output
	command.ErrWriter = &output
	err := command.Run(context.Background(), []string{"mixology-tui", "--help"})
	testutil.Ok(t, err)
	testutil.StringContains(t, output.String(), "Interactive terminal client for Mixology")
}

func TestDefaultLogPathFollowsDatabase(t *testing.T) {
	t.Parallel()

	testutil.Equals(t, defaultLogPath("data/mixology.db"), "data/mixology-tui.log")
	testutil.Equals(t, defaultLogPath("mixology.db"), "mixology-tui.log")
}
