package tui

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	auditui "github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces/tui"
	drinksui "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/tui"
	ingredientsui "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/tui"
	inventoryui "github.com/TheFellow/go-modular-monolith/app/domains/inventory/surfaces/tui"
	menusui "github.com/TheFellow/go-modular-monolith/app/domains/menus/surfaces/tui"
	ordersui "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

// TestRootViewModelConformance is the standard-view compatibility suite. A new
// root ViewModel belongs here so sizing, rendering, input ownership, and help
// metadata cannot silently diverge from the application contract.
func TestRootViewModelConformance(t *testing.T) {
	t.Parallel()

	factories := map[string]func(*app.Session) views.ViewModel{
		"audit":       func(a *app.Session) views.ViewModel { return auditui.NewListViewModel(a) },
		"dashboard":   func(a *app.Session) views.ViewModel { return views.NewDashboard(a) },
		"drinks":      func(a *app.Session) views.ViewModel { return drinksui.NewListViewModel(a) },
		"ingredients": func(a *app.Session) views.ViewModel { return ingredientsui.NewListViewModel(a) },
		"inventory":   func(a *app.Session) views.ViewModel { return inventoryui.NewListViewModel(a) },
		"menus":       func(a *app.Session) views.ViewModel { return menusui.NewListViewModel(a) },
		"orders":      func(a *app.Session) views.ViewModel { return ordersui.NewListViewModel(a) },
		"tags":        func(a *app.Session) views.ViewModel { return views.NewTags(a) },
	}

	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fixture := testutil.NewFixture(t)
			model := factory(fixture.App)

			for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 120, Height: 40}, {Width: 200, Height: 60}} {
				var cmd tea.Cmd
				model, cmd = model.Update(size)
				_ = cmd
				if model.View() == "" {
					t.Fatalf("View() is empty at %dx%d", size.Width, size.Height)
				}
				assertInteractionContract(t, model)
				assertHelpContract(t, model)
			}
		})
	}
}

func assertInteractionContract(t testing.TB, model views.ViewModel) {
	t.Helper()
	interaction := model.Interaction()
	if interaction.CapturesText && !interaction.HandlesBack {
		t.Fatal("a text-capturing view must own Back so it can dismiss its local input before navigation")
	}
}

func assertHelpContract(t testing.TB, model views.ViewModel) {
	t.Helper()
	groups := model.FullHelp()
	if len(groups) == 0 {
		t.Fatal("FullHelp() must describe the actions available in the current state")
	}
	for _, group := range groups {
		for _, binding := range group {
			help := binding.Help()
			if len(binding.Keys()) == 0 || help.Key == "" || help.Desc == "" {
				t.Fatalf("incomplete help binding: keys=%v help=%+v", binding.Keys(), help)
			}
		}
	}
}
