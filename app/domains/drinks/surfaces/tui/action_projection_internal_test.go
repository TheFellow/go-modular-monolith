//nolint:paralleltest // projector overrides are local to each view model.
package tui

import (
	"context"
	"testing"

	application "github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	drinksauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDrinkActionsProjectIntoTUIKeysAndHandlers(t *testing.T) {
	fix := testutil.NewFixture(t)
	readOnly := application.NewSession(fix.ActorContext("anonymous"), fix.App.App)
	vm := NewListViewModel(readOnly)
	drink := models.Drink{ID: models.NewDrinkID("read-only-drink"), Name: "Read only"}
	vm.list.SetItems([]list.Item{newDrinkItem(drink)})
	vm.syncDetail()
	vm.syncActions()

	testutil.Equals(t, vm.actionEnabled(drinks.ControlList), true)
	for _, id := range []string{"c", "e", "d", "t"} {
		for _, binding := range vm.ShortHelp() {
			testutil.ErrorIf(t, binding.Help().Key == id, "unauthorized help exposed %q", id)
		}
	}
	for _, key := range []string{"c", "e", "d", "t", "enter"} {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		if key == "enter" {
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		}
		vm.Update(msg)
		testutil.Equals(t, vm.mode, listModeBrowsing)
	}
}

func TestListEntryDoesNotAuthorizeAgainstSyntheticDrink(t *testing.T) {
	fix := testutil.NewFixture(t)
	vm := NewListViewModel(fix.App)
	vm.projector = drinks.ActionProjector{Authorize: func(_ context.Context, _ cedar.EntityUID, action cedar.EntityUID, _ cedar.Entity) error {
		if action == drinksauthz.ActionList {
			return errors.Permissionf("list denied")
		}
		return nil
	}}
	vm.syncActions()
	testutil.Equals(t, vm.actionEnabled(drinks.ControlList), true)
	foundRefresh := false
	for _, binding := range vm.ShortHelp() {
		foundRefresh = foundRefresh || binding.Help().Key == "r"
	}
	testutil.Equals(t, foundRefresh, true)
}

func TestActionProjectionEvaluatorErrorRecovers(t *testing.T) {
	fix := testutil.NewFixture(t)
	want := errors.New("policy evaluator unavailable")
	failing := true
	vm := NewListViewModel(fix.App)
	vm.projector = drinks.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error {
		if failing {
			return want
		}
		return nil
	}}
	vm.syncActions()
	testutil.ErrorIf(t, !errors.Is(vm.err, want), "projection error = %v, want %v", vm.err, want)
	testutil.Equals(t, len(vm.actions), 0)

	failing = false
	vm.syncActions()
	testutil.Ok(t, vm.err)
	testutil.Equals(t, vm.actionEnabled(drinks.ControlList), true)
	testutil.Equals(t, vm.actionEnabled(drinks.ControlCreate), true)
}
