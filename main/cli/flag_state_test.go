package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/urfave/cli/v3"
)

func TestIndependentCommandOutputFlags(t *testing.T) {
	t.Parallel()

	for i := range 8 {
		t.Run(fmt.Sprintf("client-%d", i), func(t *testing.T) {
			t.Parallel()

			fixture := newCLIE2E(t.TempDir() + "/flags.db")
			structured := fixture.Run("ingredients", "list", "--json")
			testutil.Ok(t, structured.Err)
			testutil.IsTrue(t, json.Valid([]byte(structured.Stdout)))

			plain := fixture.Run("ingredients", "list")
			testutil.Ok(t, plain.Err)
			testutil.StringContains(t, plain.Stdout, "DESCRIPTION")
			testutil.IsFalse(t, json.Valid([]byte(plain.Stdout)))
		})
	}
}

func TestIndependentCommandsDoNotRetainExplicitFlags(t *testing.T) {
	t.Parallel()

	for _, explicit := range []bool{true, false} {
		client, err := NewCLI()
		testutil.Ok(t, err)
		command := client.Command()
		// Exercise the real command tree and flag parser without opening storage.
		command.Before, command.After = nil, nil
		command.Command("menus").Command("list").Action = func(_ context.Context, parsed *cli.Command) error {
			for _, name := range []string{"json", "costs", "target-margin"} {
				testutil.Equals(t, parsed.IsSet(name), explicit)
			}
			testutil.Equals(t, parsed.Bool("json"), explicit)
			testutil.Equals(t, parsed.Bool("costs"), explicit)
			margin := 0.7
			if explicit {
				margin = 0.4
			}
			testutil.Equals(t, parsed.Float64("target-margin"), margin)
			return nil
		}
		args := []string{"mixology", "menus", "list"}
		if explicit {
			args = append(args, "--json", "--costs", "--target-margin", "0.4")
		}
		testutil.Ok(t, command.Run(context.Background(), args))
	}
}
