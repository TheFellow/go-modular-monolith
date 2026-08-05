//nolint:paralleltest // projector overrides are local to each view model.
package tui

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"

	application "github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestIngredientActionsProjectIntoTUIKeysAndHandlers(t *testing.T) {
	fix := testutil.NewFixture(t)
	readOnly := application.NewSession(fix.ActorContext("anonymous"), fix.App.App)
	vm := NewListViewModel(readOnly)
	ingredient := models.Ingredient{ID: entity.NewIngredientID(), Name: "Read only"}
	vm.shell.SetResult([]list.Item{newIngredientItem(ingredient)}, nil)
	vm.syncDetail()
	vm.syncActions()

	testutil.Equals(t, vm.actionEnabled(ingredients.ControlList), true)
	for _, id := range []string{"c", "e", "d", "t"} {
		for _, binding := range vm.ShortHelp() {
			testutil.ErrorIf(t, binding.Help().Key == id, "unauthorized help exposed %q", id)
		}
	}
	for _, pressed := range []string{"c", "e", "d", "t", "enter"} {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(pressed)}
		if pressed == "enter" {
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		}
		vm.Update(msg)
		testutil.Equals(t, vm.mode, listModeBrowsing)
	}
}

func TestDeniedIngredientListCapabilitySuppressesLoadingAndNavigation(t *testing.T) {
	fix := testutil.NewFixture(t)
	vm := NewListViewModel(fix.App)
	vm.projector = ingredients.ActionProjector{Authorize: func(_ context.Context, _ cedar.EntityUID, action cedar.EntityUID, _ cedar.Entity) error {
		if action == ingredientauthz.ActionList {
			return apperrors.Permissionf("list denied")
		}
		return nil
	}}
	vm.syncActions()
	testutil.ErrorIf(t, vm.Init() != nil, "%v", "denied list capability started a load")
	for _, binding := range vm.ShortHelp() {
		testutil.ErrorIf(t, binding.Help().Key == "r", "%v", "denied list capability exposed refresh")
	}
	vm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	testutil.Equals(t, vm.shell.Loading(), false)
	vm.next = "next-page"
	vm.history = []paging.Cursor{"previous-page"}
	for _, pressed := range []string{"f", "]", "["} {
		vm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(pressed)})
		testutil.Equals(t, vm.mode, listModeBrowsing)
		testutil.Equals(t, vm.request.Cursor, paging.Cursor(""))
	}
}

func TestIngredientActionProjectionEvaluatorErrorRecovers(t *testing.T) {
	fix := testutil.NewFixture(t)
	want := stderrors.New("policy evaluator unavailable")
	failing := true
	vm := NewListViewModel(fix.App)
	vm.projector = ingredients.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error {
		if failing {
			return want
		}
		return nil
	}}
	vm.syncActions()
	testutil.ErrorIf(t, !stderrors.Is(vm.actionErr, want), "projection error = %v", vm.actionErr)
	testutil.ErrorIf(t, !strings.Contains(vm.View(), want.Error()), "projection error not rendered: %s", vm.View())
	testutil.Equals(t, len(vm.actions), 0)

	failing = false
	vm.syncActions()
	testutil.Ok(t, vm.actionErr)
	testutil.Equals(t, vm.actionEnabled(ingredients.ControlList), true)
	testutil.Equals(t, vm.actionEnabled(ingredients.ControlCreate), true)
	testutil.ErrorIf(t, strings.Contains(vm.View(), want.Error()), "recovered projection still renders error: %s", vm.View())
}
