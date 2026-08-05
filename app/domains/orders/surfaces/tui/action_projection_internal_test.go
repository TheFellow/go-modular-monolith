//nolint:paralleltest // projector overrides are local to each view model.
package tui

import (
	"context"
	stderrors "errors"
	"testing"

	application "github.com/TheFellow/go-modular-monolith/app"
	orders "github.com/TheFellow/go-modular-monolith/app/domains/orders"
	ordersauthz "github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
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

func TestListEntryDoesNotAuthorizeAgainstSyntheticOrder(t *testing.T) {
	fix := testutil.NewFixture(t)
	vm := NewListViewModel(fix.App)
	vm.projector = orders.ActionProjector{Authorize: func(_ context.Context, _ cedar.EntityUID, action cedar.EntityUID, _ cedar.Entity) error {
		if action == ordersauthz.ActionList {
			return apperrors.Permissionf("denied")
		}
		return nil
	}}
	vm.syncActions()
	testutil.ErrorIf(t, !vm.actionEnabled(orders.ControlList), "list entry should remain public")
	testutil.ErrorIf(t, !vm.actionEnabled(orders.ControlPlace), "list denial disabled placement")
	foundRefresh := false
	for _, binding := range vm.ShortHelp() {
		help := binding.Help()
		foundRefresh = foundRefresh || help.Key == vm.keys.Refresh.Help().Key
	}
	testutil.ErrorIf(t, !foundRefresh, "public list omitted refresh help")
}

func TestOrderProjectionEvaluatorErrorsSurfaceInTUI(t *testing.T) {
	fix := testutil.NewFixture(t)
	want := stderrors.New("policy evaluator unavailable")
	businessErr := stderrors.New("order load failed")
	vm := NewListViewModel(fix.App)
	vm.err = businessErr
	failing := true
	vm.projector = orders.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error {
		if failing {
			return want
		}
		return nil
	}}
	vm.syncActions()
	testutil.ErrorIf(t, !stderrors.Is(vm.actionErr, want), "projection error = %v", vm.actionErr)
	testutil.Equals(t, len(vm.actions), 0)
	failing = false
	vm.syncActions()
	testutil.ErrorIf(t, vm.actionErr != nil, "recovered projection error = %v", vm.actionErr)
	testutil.ErrorIf(t, !stderrors.Is(vm.err, businessErr), "projection recovery cleared business error: %v", vm.err)
	testutil.Equals(t, vm.actionEnabled(orders.ControlList), true)
}
