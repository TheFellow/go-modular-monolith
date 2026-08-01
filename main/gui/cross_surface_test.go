package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	auditgui "github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces/gui"
	ingredientauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	ingredientsgui "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/gui"
	inventoryauthz "github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	inventorygui "github.com/TheFellow/go-modular-monolith/app/domains/inventory/surfaces/gui"
	tagginggui "github.com/TheFellow/go-modular-monolith/app/domains/tagging/surfaces/gui"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
)

//nolint:paralleltest // exercises independent process lifecycles against one database file.
func TestCLIAndComposedDesktopShareIngredientInventoryAuditAndTagContracts(t *testing.T) {
	repository, err := filepath.Abs("../..")
	testutil.ErrorIf(t, err != nil, "%v", err)
	workingDirectory := t.TempDir()
	binary := testutil.ExecutablePath(workingDirectory, "mixology")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "./main/cli")
	build.Dir = repository
	{
		output, buildErr := build.CombinedOutput()
		testutil.ErrorIf(t, buildErr != nil, "build CLI: %v\n%s", buildErr, output)
	}
	run := func(args ...string) string {
		t.Helper()
		command := exec.CommandContext(t.Context(), binary, append([]string{"--log-level", "error"}, args...)...)
		command.Dir = workingDirectory
		output, runErr := command.CombinedOutput()
		testutil.ErrorIf(t, runErr != nil, "CLI %v: %v\n%s", args, runErr, output)
		return string(output)
	}

	ingredientID := strings.TrimSpace(run("ingredients", "create", "--category", "other", "--unit", "oz", "Lifecycle ingredient"))
	run("inventory", "set", "--ingredient-id", ingredientID, "--quantity", "12", "--cost-per-unit", "$1.25")
	run("tags", "add", ingredientID, "origin=cli")

	gui := test.NewApp()
	desktop, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{
		dataDirectory: filepath.Join(workingDirectory, "data"), actor: "owner",
	}, deterministicDesktopDependencies(nil))
	testutil.ErrorIf(t, err != nil, "%v", err)
	driver := fynetest.NewDriver(t, desktop.shell.Content())
	{
		err := desktop.shell.Navigate("ingredients")
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	driver.Tap("ingredients-refresh")
	ingredients := desktop.presenters["ingredients"].(*ingredientsgui.Presenter).Snapshot()
	testutil.ErrorIf(t, len(ingredients.Items) != 1 || ingredients.Items[0].Name != "Lifecycle ingredient", "Fyne did not observe CLI ingredient: %#v", ingredients)

	{
		err := desktop.shell.Navigate("inventory")
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	driver.Tap("inventory-refresh")
	inventory := desktop.presenters["inventory"].(*inventorygui.Presenter).Snapshot()
	testutil.ErrorIf(t, len(inventory.Rows) != 1 || inventory.Rows[0].Inventory.IngredientID.String() != ingredientID, "Fyne did not observe CLI inventory: %#v", inventory)

	{
		err := desktop.shell.Navigate("audit")
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	driver.Tap(auditgui.ControlRefresh)
	audit := desktop.presenters["audit"].(*auditgui.Presenter).State()
	testutil.ErrorIf(t, audit.Err != nil, "Fyne did not observe CLI audit history: %#v", audit)
	wantActions := map[string]bool{
		ingredientauthz.ActionCreate.String(): false,
		inventoryauthz.ActionSet.String():     false,
		ingredientauthz.ActionTag.String():    false,
	}
	for _, row := range audit.Rows {
		if _, wanted := wantActions[row.Entry.Action]; wanted && row.Entry.Success {
			wantActions[row.Entry.Action] = true
		}
	}
	for action, found := range wantActions {
		testutil.ErrorIf(t, !found, "Fyne audit did not contain successful CLI action %s: %#v", action, audit.Rows)
	}

	{
		err := desktop.shell.Navigate("tags")
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	driver.Tap(tagginggui.ControlInspect)
	driver.Tap("tags.type.Mixology::Ingredient")
	driver.Tap("tags.entity.0")
	tagState := desktop.presenters["tags"].(*tagginggui.Presenter).State()
	testutil.ErrorIf(t, tagState.Result.Tags.Canonical().String() != "origin=cli", "Fyne did not observe CLI tags: %#v", tagState)
	driver.Tap(tagginggui.ControlBack)
	driver.Tap(tagginggui.ControlAdd)
	driver.Tap("tags.type.Mixology::Ingredient")
	driver.Tap("tags.entity.0")
	driver.Type(tagginggui.ControlValue, "origin=fyne")
	driver.Tap(tagginggui.ControlSubmit)
	{
		state := desktop.presenters["tags"].(*tagginggui.Presenter).State()
		testutil.ErrorIf(t, state.Err != nil || !state.Result.Changed, "Fyne tag mutation failed: %#v", state)
	}
	{
		err := desktop.Close()
		testutil.ErrorIf(t, err != nil, "%v", err)
	}
	gui.Quit()

	output := run("tags", "list", ingredientID)
	testutil.ErrorIf(t, !strings.Contains(output, "origin=fyne"), "CLI did not observe Fyne tag after a fresh lifecycle:\n%s", output)
}
