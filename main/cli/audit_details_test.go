//nolint:paralleltest // CLI integration exercises sequential database lifecycles.
package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestAuditDetailsAvailableAcrossListHistoryAndActor(t *testing.T) {
	cli := newCLIE2E(filepath.Join(t.TempDir(), "audit.db"))
	created := cli.Run("ingredients", "create", "Lime", "--category", "juice", "--unit", "oz")
	testutil.Ok(t, created.Err)
	id := strings.TrimSpace(created.Stdout)
	retired := cli.Run("ingredients", "retire", "--id", id, "--reason", "discontinued")
	testutil.Ok(t, retired.Err)
	for _, args := range [][]string{{"audit", "list"}, {"audit", "history", "Mixology::Ingredient::" + id}, {"audit", "actor", "owner"}} {
		result := cli.Run(append(args, "--details")...)
		testutil.Ok(t, result.Err)
		testutil.StringContains(t, result.Stdout, "Referenced entities")
		testutil.StringContains(t, result.Stdout, "Committed effects")
		testutil.StringContains(t, result.Stdout, "Before: ")
		testutil.StringContains(t, result.Stdout, "After: ")
		compact := cli.Run(args...)
		testutil.Ok(t, compact.Err)
		testutil.ErrorIf(t, strings.Contains(compact.Stdout, "Committed effects"), "compact audit table expanded unexpectedly")
		structured := cli.Run(append(args, "--details", "--json")...)
		testutil.Ok(t, structured.Err)
		testutil.ErrorIf(t, strings.Contains(structured.Stdout, "Committed effects"), "human detail text corrupted JSON output")
		testutil.StringContains(t, structured.Stdout, `"Effects"`)
	}
}
