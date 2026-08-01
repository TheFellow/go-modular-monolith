//nolint:paralleltest // fresh-process integration tests deliberately serialize database lifecycles.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	menucli "github.com/TheFellow/go-modular-monolith/app/domains/menus/surfaces/cli"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestMenusCLIUpdateDeleteLifecycleAndCrossInvocationVisibility(t *testing.T) {
	cli := newCLIE2E(filepath.Join(t.TempDir(), "menus.db"))
	created := cli.Run("menus", "create", "Dinner", "--json")
	testutil.Ok(t, created.Err)
	var menu menucli.Menu
	testutil.Ok(t, json.Unmarshal([]byte(created.Stdout), &menu))

	updated := cli.Run("menus", "update", "--id", menu.ID, "--name", "Late Dinner", "--description", "After hours", "--tags", "featured,service=late", "--json")
	testutil.Ok(t, updated.Err)
	testutil.Ok(t, json.Unmarshal([]byte(updated.Stdout), &menu))
	testutil.Equals(t, menu.Name, "Late Dinner")
	testutil.Equals(t, menu.Description, "After hours")
	testutil.Equals(t, strings.Join(menu.Tags, ","), "featured,service=late")

	// A fresh command tree observes the same persisted state, as the TUI and
	// GUI surfaces do through the shared application modules.
	shown := cli.Run("menus", "show", "--id", menu.ID, "--json")
	testutil.Ok(t, shown.Err)
	testutil.StringContains(t, shown.Stdout, `"name": "Late Dinner"`)
	testutil.StringContains(t, shown.Stdout, `"description": "After hours"`)

	// The public update API treats an empty description as preserve, and the
	// CLI exposes that behavior rather than claiming it clears the field.
	preserved := cli.Run("menus", "update", "--id", menu.ID, "--description=", "--json")
	testutil.Ok(t, preserved.Err)
	testutil.StringContains(t, preserved.Stdout, `"description": "After hours"`)

	deleted := cli.Run("menus", "delete", "--id", menu.ID, "--json")
	testutil.Ok(t, deleted.Err)
	testutil.StringContains(t, deleted.Stdout, `"status": "archived"`)
	missing := cli.Run("menus", "show", "--id", menu.ID)
	testutil.ErrorIf(t, missing.Err == nil, "%v", "deleted menu remained visible")
}

func TestMenusCLIUpdateStructuredInputAndTemplate(t *testing.T) {
	dir := t.TempDir()
	cli := newCLIE2E(filepath.Join(dir, "menus.db"))
	created := cli.Run("menus", "create", "Patio", "--json")
	testutil.Ok(t, created.Err)
	var menu menucli.Menu
	testutil.Ok(t, json.Unmarshal([]byte(created.Stdout), &menu))
	input := filepath.Join(dir, "update.json")
	testutil.Ok(t, os.WriteFile(input, []byte(`{"id":"`+menu.ID+`","name":"Garden Patio","description":"Outside"}`), 0o600))
	updated := cli.Run("menus", "update", "--file", input, "--json")
	testutil.Ok(t, updated.Err)
	testutil.StringContains(t, updated.Stdout, `"name": "Garden Patio"`)
	template := cli.Run("menus", "update", "--template")
	testutil.Ok(t, template.Err)
	testutil.StringContains(t, template.Stdout, `"id": "mnu-..."`)
}

func TestMenusCLIAnalysisRejectsNonFiniteAndOutOfRangeMargins(t *testing.T) {
	cli := newCLIE2E(filepath.Join(t.TempDir(), "menus.db"))
	for _, margin := range []string{"NaN", "+Inf", "0", "1"} {
		result := cli.Run("menus", "list", "--costs", "--target-margin", margin)
		testutil.ErrorIf(t, result.Err == nil, "target margin %q was accepted", margin)
		testutil.StringContains(t, result.Err.Error(), "target margin must be a number between 0 and 1")
	}
}

func TestMenusCLIUpdateAndDeleteEnforceValidationAuthorizationStateAndAtomicTags(t *testing.T) {
	cli := newCLIE2E(filepath.Join(t.TempDir(), "menus.db"))
	created := cli.Run("menus", "create", "Original", "--tags", "stable=yes", "--json")
	testutil.Ok(t, created.Err)
	var menu menucli.Menu
	testutil.Ok(t, json.Unmarshal([]byte(created.Stdout), &menu))

	invalid := cli.Run("menus", "update", "--id", menu.ID, "--name", "Must Not Persist", "--tags", "region=east,region=west")
	testutil.ErrorIf(t, invalid.Err == nil, "%v", "invalid duplicate tags were accepted")
	shown := cli.Run("menus", "show", "--id", menu.ID, "--json")
	testutil.Ok(t, shown.Err)
	testutil.StringContains(t, shown.Stdout, `"name": "Original"`)
	testutil.StringContains(t, shown.Stdout, `"stable=yes"`)

	denied := cli.As("bartender").Run("menus", "update", "--id", menu.ID, "--name", "Denied")
	testutil.ErrorIf(t, denied.Err == nil, "%v", "unauthorized update was accepted")
	deniedDelete := cli.As("bartender").Run("menus", "delete", "--id", menu.ID)
	testutil.ErrorIf(t, deniedDelete.Err == nil, "%v", "unauthorized delete was accepted")

	testutil.Ok(t, cli.Run("menus", "publish", "--id", menu.ID).Err)
	for _, result := range []cliResult{
		cli.Run("menus", "update", "--id", menu.ID, "--name", "Published Change"),
		cli.Run("menus", "delete", "--id", menu.ID),
	} {
		testutil.ErrorIf(t, result.Err == nil, "%v", "published menu mutation was accepted")
	}
	shown = cli.Run("menus", "show", "--id", menu.ID, "--json")
	testutil.Ok(t, shown.Err)
	testutil.StringContains(t, shown.Stdout, `"name": "Original"`)
	testutil.StringContains(t, shown.Stdout, `"status": "published"`)
}

func TestMenusCLIUpdateAndDeleteEmitTouchedAuditEntries(t *testing.T) {
	cli := newCLIE2E(filepath.Join(t.TempDir(), "menus.db"))
	created := cli.Run("menus", "create", "Audited", "--json")
	testutil.Ok(t, created.Err)
	var menu menucli.Menu
	testutil.Ok(t, json.Unmarshal([]byte(created.Stdout), &menu))
	testutil.Ok(t, cli.Run("menus", "update", "--id", menu.ID, "--name", "Audited Update").Err)
	updatedAudit := cli.Run("audit", "list", "--entity", "Mixology::Menu::"+menu.ID, "--json")
	testutil.Ok(t, updatedAudit.Err)
	requireAuditAction(t, updatedAudit.Stdout, "update")
	testutil.Ok(t, cli.Run("menus", "delete", "--id", menu.ID).Err)
	deletedAudit := cli.Run("audit", "list", "--entity", "Mixology::Menu::"+menu.ID, "--json")
	testutil.Ok(t, deletedAudit.Err)
	requireAuditAction(t, deletedAudit.Stdout, "delete")
}

func requireAuditAction(t *testing.T, output, action string) {
	t.Helper()
	var page struct {
		Items []struct {
			Action string
		}
	}
	testutil.Ok(t, json.Unmarshal([]byte(output), &page))
	for _, item := range page.Items {
		if strings.Contains(item.Action, `Menu::Action::"`+action+`"`) {
			return
		}
	}
	testutil.Fail(t, "audit action %q not found in %s", action, output)
}
