package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	auditfyne "github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces/fyne"
	ingredientauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	ingredientsfyne "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/fyne"
	inventoryauthz "github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	inventoryfyne "github.com/TheFellow/go-modular-monolith/app/domains/inventory/surfaces/fyne"
	taggingfyne "github.com/TheFellow/go-modular-monolith/app/domains/tagging/surfaces/fyne"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
)

//nolint:paralleltest // exercises independent process lifecycles against one database file.
func TestCLIAndComposedDesktopShareIngredientInventoryAuditAndTagContracts(t *testing.T) {
	repository, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	binary := filepath.Join(workingDirectory, "mixology")
	build := exec.Command("go", "build", "-o", binary, "./main/cli")
	build.Dir = repository
	if output, buildErr := build.CombinedOutput(); buildErr != nil {
		t.Fatalf("build CLI: %v\n%s", buildErr, output)
	}
	run := func(args ...string) string {
		t.Helper()
		command := exec.Command(binary, append([]string{"--log-level", "error"}, args...)...)
		command.Dir = workingDirectory
		output, runErr := command.CombinedOutput()
		if runErr != nil {
			t.Fatalf("CLI %v: %v\n%s", args, runErr, output)
		}
		return string(output)
	}

	ingredientID := strings.TrimSpace(run("ingredients", "create", "--category", "other", "--unit", "oz", "Lifecycle ingredient"))
	run("inventory", "set", "--ingredient-id", ingredientID, "--quantity", "12", "--cost-per-unit", "$1.25")
	run("tags", "add", ingredientID, "origin=cli")

	gui := test.NewApp()
	desktop, err := openDesktopWithDependencies(context.Background(), gui, desktopConfig{
		dataDirectory: filepath.Join(workingDirectory, "data"), actor: "owner",
	}, deterministicDesktopDependencies(nil))
	if err != nil {
		t.Fatal(err)
	}
	driver := fynetest.NewDriver(t, desktop.shell.Content())
	if err := desktop.shell.Navigate("ingredients"); err != nil {
		t.Fatal(err)
	}
	driver.Tap("ingredients-refresh")
	ingredients := desktop.presenters["ingredients"].(*ingredientsfyne.Presenter).Snapshot()
	if len(ingredients.Items) != 1 || ingredients.Items[0].Name != "Lifecycle ingredient" {
		t.Fatalf("Fyne did not observe CLI ingredient: %#v", ingredients)
	}

	if err := desktop.shell.Navigate("inventory"); err != nil {
		t.Fatal(err)
	}
	driver.Tap("inventory-refresh")
	inventory := desktop.presenters["inventory"].(*inventoryfyne.Presenter).Snapshot()
	if len(inventory.Rows) != 1 || inventory.Rows[0].Inventory.IngredientID.String() != ingredientID {
		t.Fatalf("Fyne did not observe CLI inventory: %#v", inventory)
	}

	if err := desktop.shell.Navigate("audit"); err != nil {
		t.Fatal(err)
	}
	driver.Tap(auditfyne.ControlRefresh)
	audit := desktop.presenters["audit"].(*auditfyne.Presenter).State()
	if audit.Err != nil {
		t.Fatalf("Fyne did not observe CLI audit history: %#v", audit)
	}
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
		if !found {
			t.Fatalf("Fyne audit did not contain successful CLI action %s: %#v", action, audit.Rows)
		}
	}

	if err := desktop.shell.Navigate("tags"); err != nil {
		t.Fatal(err)
	}
	driver.Tap(taggingfyne.ControlInspect)
	driver.Tap("tags.type.Mixology::Ingredient")
	driver.Tap("tags.entity.0")
	tagState := desktop.presenters["tags"].(*taggingfyne.Presenter).State()
	if tagState.Result.Tags.Canonical().String() != "origin=cli" {
		t.Fatalf("Fyne did not observe CLI tags: %#v", tagState)
	}
	driver.Tap(taggingfyne.ControlBack)
	driver.Tap(taggingfyne.ControlAdd)
	driver.Tap("tags.type.Mixology::Ingredient")
	driver.Tap("tags.entity.0")
	driver.Type(taggingfyne.ControlValue, "origin=fyne")
	driver.Tap(taggingfyne.ControlSubmit)
	if state := desktop.presenters["tags"].(*taggingfyne.Presenter).State(); state.Err != nil || !state.Result.Changed {
		t.Fatalf("Fyne tag mutation failed: %#v", state)
	}
	if err := desktop.Close(); err != nil {
		t.Fatal(err)
	}
	gui.Quit()

	output := run("tags", "list", ingredientID)
	if !strings.Contains(output, "origin=fyne") {
		t.Fatalf("CLI did not observe Fyne tag after a fresh lifecycle:\n%s", output)
	}
}
