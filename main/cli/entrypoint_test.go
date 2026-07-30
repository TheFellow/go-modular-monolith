package main

import (
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestCLIRejectsFormerTUIFlag(t *testing.T) {
	t.Parallel()

	application, err := NewCLI()
	testutil.Ok(t, err)
	err = application.Command().Run(context.Background(), []string{"mixology", "--tui"})
	testutil.ErrorIf(t, err == nil, "CLI accepted removed --tui flag")
}
