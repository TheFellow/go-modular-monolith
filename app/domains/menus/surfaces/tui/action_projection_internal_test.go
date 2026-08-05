//nolint:paralleltest // projector overrides are local to each view model.
package tui

import (
	"context"
	stderrors "errors"
	"testing"

	application "github.com/TheFellow/go-modular-monolith/app"
	menus "github.com/TheFellow/go-modular-monolith/app/domains/menus"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuActionsProjectIntoTUIKeysAndDisabledDetail(t *testing.T) {
	fix := testutil.NewFixture(t)
	drinkID := entity.DrinkID(cedar.NewEntityUID(entity.TypeDrink, "projection-drink"))
	tests := []struct {
		name           string
		menu           models.Menu
		publish, draft bool
		publishReason  string
		removeReason   string
	}{
		{name: "draft", menu: models.Menu{ID: models.NewMenuID("projection-draft"), Name: "Draft", Status: models.MenuStatusDraft, Items: []models.MenuItem{{DrinkID: drinkID}}}, publish: true},
		{name: "published", menu: models.Menu{ID: models.NewMenuID("projection-published"), Name: "Published", Status: models.MenuStatusPublished, Items: []models.MenuItem{{DrinkID: drinkID}}}, draft: true, publishReason: "Available only while the menu is a draft."},
		{name: "empty", menu: models.Menu{ID: models.NewMenuID("projection-empty"), Name: "Empty", Status: models.MenuStatusDraft}, publishReason: "Add at least one drink before publishing.", removeReason: "Add a drink before trying to remove one."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vm := NewListViewModel(fix.App)
			vm.list.SetItems([]list.Item{newMenuItem(tc.menu, vm.styles)})
			vm.syncDetail()
			vm.syncActions()
			testutil.Equals(t, vm.actionEnabled(menus.ControlPublish), tc.publish)
			testutil.Equals(t, vm.actionEnabled(menus.ControlDraft), tc.draft)
			view := vm.detail.View()
			if tc.publishReason != "" {
				testutil.StringContains(t, view, tc.publishReason)
			}
			if tc.removeReason != "" {
				testutil.StringContains(t, view, tc.removeReason)
			}
		})
	}
}

func TestUnauthorizedMenuKeysAndHelpAreOmitted(t *testing.T) {
	fix := testutil.NewFixture(t)
	readOnly := application.NewSession(fix.ActorContext("sommelier"), fix.App.App)
	vm := NewListViewModel(readOnly)
	menu := models.Menu{ID: models.NewMenuID("read-only-menu"), Name: "Read only", Status: models.MenuStatusDraft}
	vm.list.SetItems([]list.Item{newMenuItem(menu, vm.styles)})
	vm.syncDetail()
	vm.syncActions()

	for _, binding := range vm.ShortHelp() {
		help := binding.Help()
		testutil.ErrorIf(t, help.Key == "p" || help.Key == "e" || help.Key == "a", "unauthorized help exposed %q", help.Key)
	}
	vm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	testutil.Equals(t, vm.mode, listModeBrowsing)
	testutil.ErrorIf(t, vm.taggedDialog != nil, "%v", "unauthorized publish key opened confirmation")
}

func TestActionProjectionEvaluatorErrorsSurface(t *testing.T) {
	fix := testutil.NewFixture(t)
	want := stderrors.New("policy evaluator unavailable")
	vm := NewListViewModel(fix.App)
	vm.projector = menus.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error { return want }}
	vm.syncActions()
	testutil.ErrorIf(t, !stderrors.Is(vm.err, want), "projection error = %v, want %v", vm.err, want)
	testutil.Equals(t, len(vm.actions), 0)
	for _, binding := range vm.ShortHelp() {
		help := binding.Help()
		testutil.ErrorIf(t, help.Key == "c", "%v", "failed projection exposed create help")
	}
}
