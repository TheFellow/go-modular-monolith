//nolint:paralleltest // projector overrides are local to each view model.
package tui

import (
	"context"
	stderrors "errors"
	"testing"

	application "github.com/TheFellow/go-modular-monolith/app"
	inventory "github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

func TestUnauthorizedInventoryKeysAndHelpAreOmitted(t *testing.T) {
	fix := testutil.NewFixture(t)
	readOnly := application.NewSession(fix.ActorContext("sommelier"), fix.App.App)
	vm := NewListViewModel(readOnly)
	vm.rows = []InventoryRow{{Inventory: projectedInventory()}}
	vm.table.SetRows([]table.Row{{"stock"}})
	vm.syncActions()
	for _, binding := range vm.ShortHelp() {
		help := binding.Help()
		testutil.ErrorIf(t, help.Key == "a" || help.Key == "s" || help.Key == "t", "unauthorized help exposed %q", help.Key)
	}
	vm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	testutil.Equals(t, vm.mode, listModeBrowsing)
}

func TestInventoryProjectionEvaluatorErrorsSurface(t *testing.T) {
	fix := testutil.NewFixture(t)
	want := stderrors.New("policy evaluator unavailable")
	vm := NewListViewModel(fix.App)
	vm.rows = []InventoryRow{{Inventory: projectedInventory()}}
	vm.table.SetRows([]table.Row{{"stock"}})
	failing := true
	vm.projector = inventory.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error {
		if failing {
			return want
		}
		return nil
	}}
	vm.syncActions()
	testutil.ErrorIf(t, !stderrors.Is(vm.actionErr, want), "projection error = %v, want %v", vm.actionErr, want)
	testutil.Equals(t, len(vm.actions), 0)
	loadErr := stderrors.New("load still failed")
	vm.err, failing = loadErr, false
	vm.syncActions()
	testutil.Equals(t, vm.actionErr, nil)
	testutil.ErrorIf(t, !stderrors.Is(vm.err, loadErr), "load error was clobbered: %v", vm.err)
	testutil.Equals(t, vm.actionEnabled(inventory.ControlList), true)
}

func projectedInventory() models.Inventory {
	return models.Inventory{ID: entity.NewInventoryID(), IngredientID: entity.NewIngredientID(), Amount: measurement.MustAmount(1, measurement.UnitOz)}
}
