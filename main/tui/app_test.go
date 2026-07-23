package tui

import (
	"testing"

	auditui "github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces/tui"
	drinksui "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/tui"
	ingredientsui "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/tui"
	inventoryui "github.com/TheFellow/go-modular-monolith/app/domains/inventory/surfaces/tui"
	menusmodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	menusui "github.com/TheFellow/go-modular-monolith/app/domains/menus/surfaces/tui"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	ordersui "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	tea "github.com/charmbracelet/bubbletea"
)

func TestStatusBarView_UsesWarningStyleForNotFound(t *testing.T) {
	t.Parallel()

	app := &App{styles: styles.App}
	app.lastError = errors.NotFoundf("ingredient missing")

	expected := app.styles.StatusBar.Render(app.styles.WarningText.Render(app.lastError.Error()))
	testutil.Equals(t, app.statusBarView(), expected)
}

func TestStatusBarView_UsesErrorStyleForInvalid(t *testing.T) {
	t.Parallel()

	app := &App{styles: styles.App}
	app.lastError = errors.Invalidf("invalid input")

	expected := app.styles.StatusBar.Render(app.styles.ErrorText.Render(app.lastError.Error()))
	testutil.Equals(t, app.statusBarView(), expected)
}

func TestStatusBarView_UsesErrorStyleForPermission(t *testing.T) {
	t.Parallel()

	app := &App{styles: styles.App}
	app.lastError = errors.Permissionf("permission denied")

	expected := app.styles.StatusBar.Render(app.styles.ErrorText.Render(app.lastError.Error()))
	testutil.Equals(t, app.statusBarView(), expected)
}

func TestBackKey_CancelsDomainLocalStateBeforeNavigating(t *testing.T) {
	t.Parallel()

	type scenario struct {
		name     string
		view     View
		model    func(*testutil.Fixture) views.ViewModel
		activate func(testing.TB, views.ViewModel) views.ViewModel
	}

	scenarios := []scenario{
		{
			name: "drinks create error",
			view: ViewDrinks,
			model: func(f *testutil.Fixture) views.ViewModel {
				f.Bootstrap().WithBasicIngredients()
				return tuitest.InitAndLoad(t, drinksui.NewListViewModel(f.App))
			},
			activate: func(t testing.TB, model views.ViewModel) views.ViewModel {
				model = updateView(t, model, keyRunes("c"))
				return updateView(t, model, submitKey())
			},
		},
		{
			name: "ingredients create error",
			view: ViewIngredients,
			model: func(f *testutil.Fixture) views.ViewModel {
				return tuitest.InitAndLoad(t, ingredientsui.NewListViewModel(f.App))
			},
			activate: func(t testing.TB, model views.ViewModel) views.ViewModel {
				model = updateView(t, model, keyRunes("c"))
				return updateView(t, model, submitKey())
			},
		},
		{
			name: "inventory adjust error",
			view: ViewInventory,
			model: func(f *testutil.Fixture) views.ViewModel {
				f.Bootstrap().WithBasicIngredients().WithStock(5)
				return tuitest.InitAndLoad(t, inventoryui.NewListViewModel(f.App))
			},
			activate: func(t testing.TB, model views.ViewModel) views.ViewModel {
				model = updateView(t, model, keyRunes("a"))
				return updateView(t, model, submitKey())
			},
		},
		{
			name: "menus create error",
			view: ViewMenus,
			model: func(f *testutil.Fixture) views.ViewModel {
				return tuitest.InitAndLoad(t, menusui.NewListViewModel(f.App))
			},
			activate: func(t testing.TB, model views.ViewModel) views.ViewModel {
				model = updateView(t, model, keyRunes("c"))
				return updateView(t, model, submitKey())
			},
		},
		{
			name: "orders cancel dialog",
			view: ViewOrders,
			model: func(f *testutil.Fixture) views.ViewModel {
				drink := f.CreateDrink("Margarita").With("Tequila", 1).Build()
				menu, err := f.Menus.Create(f.OwnerContext(), &menusmodels.Menu{Name: "Dinner"})
				testutil.Ok(t, err)
				menu, err = f.Menus.AddDrink(f.OwnerContext(), &menusmodels.MenuPatch{MenuID: menu.ID, DrinkID: drink.ID})
				testutil.Ok(t, err)
				menu, err = f.Menus.Publish(f.OwnerContext(), &menusmodels.Menu{ID: menu.ID})
				testutil.Ok(t, err)
				_, err = f.Orders.Place(f.OwnerContext(), &ordersmodels.Order{
					MenuID: menu.ID,
					Items:  []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
				})
				testutil.Ok(t, err)
				return tuitest.InitAndLoad(t, ordersui.NewListViewModel(f.App))
			},
			activate: func(t testing.TB, model views.ViewModel) views.ViewModel {
				return updateView(t, model, keyRunes("x"))
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			f := testutil.NewFixture(t)
			model := scenario.model(f)
			model = updateView(t, model, tea.WindowSizeMsg{Width: 120, Height: 40})
			model = scenario.activate(t, model)

			handler := testutil.Cast[views.BackKeyHandler](t, model)
			testutil.IsTrue(t, handler.HandleBackKey())

			app := NewApp(f.App)
			app.currentView = scenario.view
			app.prevViews = []View{ViewDashboard}
			app.views[scenario.view] = model

			app = updateAppAndRunCmds(t, app, tea.KeyMsg{Type: tea.KeyEsc})
			testutil.Equals(t, app.currentView, scenario.view)

			if handler, ok := app.views[scenario.view].(views.BackKeyHandler); ok {
				testutil.IsFalse(t, handler.HandleBackKey())
			}

			app = updateAppOnce(t, app, tea.KeyMsg{Type: tea.KeyEsc})
			testutil.Equals(t, app.currentView, ViewDashboard)
		})
	}
}

func TestBackKey_NavigatesWhenDomainHasNoLocalState(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name  string
		view  View
		model func(*testutil.Fixture) views.ViewModel
	}{
		{
			name: ViewDrinks.String(),
			view: ViewDrinks,
			model: func(f *testutil.Fixture) views.ViewModel {
				return tuitest.InitAndLoad(t, drinksui.NewListViewModel(f.App))
			},
		},
		{
			name: ViewAudit.String(),
			view: ViewAudit,
			model: func(f *testutil.Fixture) views.ViewModel {
				return tuitest.InitAndLoad(t, auditui.NewListViewModel(f.App))
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			f := testutil.NewFixture(t)
			model := scenario.model(f)

			app := NewApp(f.App)
			app.currentView = scenario.view
			app.prevViews = []View{ViewDashboard}
			app.views[scenario.view] = model

			app = updateAppOnce(t, app, tea.KeyMsg{Type: tea.KeyEsc})
			testutil.Equals(t, app.currentView, ViewDashboard)
		})
	}
}

func updateAppOnce(t testing.TB, app *App, msg tea.Msg) *App {
	t.Helper()

	model, _ := app.Update(msg)
	return testutil.Cast[*App](t, model)
}

func updateAppAndRunCmds(t testing.TB, app *App, msg tea.Msg) *App {
	t.Helper()

	model, cmd := app.Update(msg)
	updated := testutil.Cast[*App](t, model)
	for _, msg := range tuitest.RunCmds(cmd) {
		updated = updateAppAndRunCmds(t, updated, msg)
	}
	return updated
}

func updateView(t testing.TB, model views.ViewModel, msg tea.Msg) views.ViewModel {
	t.Helper()

	updated, cmd := model.Update(msg)
	for _, msg := range tuitest.RunCmds(cmd) {
		updated = updateView(t, updated, msg)
	}
	return updated
}

func keyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func submitKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyCtrlS}
}
