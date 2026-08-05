//nolint:paralleltest // projector overrides are local to each view model.
package tui

import (
	"context"
	stderrors "errors"
	"testing"

	application "github.com/TheFellow/go-modular-monolith/app"
	orders "github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestOrderLifecycleProjectsIntoTUIKeysAndDetail(t *testing.T) {
	fix := testutil.NewFixture(t)
	vm := NewListViewModel(fix.App)
	order := models.Order{ID: entity.NewOrderID(), Status: models.OrderStatusCompleted}
	vm.list.SetItems([]list.Item{newOrderItem(order, "Menu", vm.styles)})
	vm.syncDetail()
	vm.syncActions()
	testutil.ErrorIf(t, vm.actionEnabled(orders.ControlComplete), "completed order exposed complete")
	testutil.ErrorIf(t, vm.actionEnabled(orders.ControlCancel), "completed order exposed cancel")
	testutil.StringContains(t, vm.actions[orders.ControlComplete].DisabledReason, "this order is completed")
	vm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	testutil.Equals(t, vm.mode, listModeBrowsing)
}

func TestUnauthorizedOrderKeysAndHelpAreOmitted(t *testing.T) {
	fix := testutil.NewFixture(t)
	readOnly := application.NewSession(fix.ActorContext("sommelier"), fix.App.App)
	vm := NewListViewModel(readOnly)
	vm.list.SetItems([]list.Item{newOrderItem(models.Order{ID: entity.NewOrderID(), Status: models.OrderStatusPending}, "Menu", vm.styles)})
	vm.syncActions()
	for _, binding := range vm.ShortHelp() {
		help := binding.Help()
		testutil.ErrorIf(t, help.Key == "c" || help.Key == "o" || help.Key == "x" || help.Key == "t", "unauthorized help exposed %q", help.Key)
	}
	vm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	testutil.Equals(t, vm.mode, listModeBrowsing)
}

func TestOrderProjectionEvaluatorErrorsSurfaceInTUI(t *testing.T) {
	fix := testutil.NewFixture(t)
	want := stderrors.New("policy evaluator unavailable")
	vm := NewListViewModel(fix.App)
	vm.projector = orders.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error { return want }}
	vm.syncActions()
	testutil.ErrorIf(t, !stderrors.Is(vm.err, want), "projection error = %v", vm.err)
	testutil.Equals(t, len(vm.actions), 0)
}
