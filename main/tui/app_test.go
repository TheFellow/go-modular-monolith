package tui

import (
	"fmt"
	"testing"

	auditui "github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces/tui"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	drinksui "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/tui"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	ingredientsui "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/tui"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	inventoryui "github.com/TheFellow/go-modular-monolith/app/domains/inventory/surfaces/tui"
	menusui "github.com/TheFellow/go-modular-monolith/app/domains/menus/surfaces/tui"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	ordersui "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/main/tui/styles"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func TestE2E_DashboardAndTagResultTransitionsStayWithinViewport(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "E2E Tonic", Category: ingredientsmodels.CategoryMixer, Unit: measurement.UnitMl,
	})
	for i := range 10 {
		_, err := f.App.Tags.Upsert(f.OwnerContext(), ingredient.EntityUID(), tag.Tag{
			Key: fmt.Sprintf("e2e-%d", i),
		})
		testutil.Ok(t, err)
	}

	driver := tuitest.NewDriver(t, NewApp(f.App))
	const width, height = MinWidth, MinHeight
	driver.Resize(width, height)
	driver.RequireText("Dashboard", "Recent Activity")

	driver.Press("7")
	driver.Resize(100, 40)
	driver.RequireText("Mixology > Tags", "Inspect entity tags", "Tag usage summary")
	for range 5 {
		driver.Press("down")
	}
	driver.Press("enter")
	driver.RequireText("Tag usage summary", "TOTAL", "INGREDIENTS", "e2e-0")

	driver.Press("esc")
	driver.RequireText("Inspect entity tags", "Show exact tag")
	for range 2 {
		driver.Press("up")
	}
	driver.Press("enter")
	driver.Press("e2e-0")
	driver.Press("ctrl+s")
	driver.RequireText("Show exact tag", "ENTITY TYPE", ingredient.ID.String(), "e2e-0")

	driver.Press("esc")
	driver.RequireText("Inspect entity tags")
	driver.Press("esc")
	driver.RequireText("Mixology > Dashboard")
}

func TestTagHotkeyManagesEveryOperationalEntity(t *testing.T) {
	t.Parallel()

	type scenario struct {
		name  string
		model func(testing.TB, *testutil.Fixture) (views.ViewModel, cedar.EntityUID)
	}

	scenarios := []scenario{
		{name: "drink", model: func(t testing.TB, f *testutil.Fixture) (views.ViewModel, cedar.EntityUID) {
			ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Lime", Category: ingredientsmodels.CategoryJuice, Unit: measurement.UnitOz})
			drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
				Name: "Daiquiri", Category: drinksmodels.DrinkCategoryCocktail,
				Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, ingredient.Unit)}}, Steps: []string{"Shake"}},
			})
			return tuitest.InitAndLoad(t, drinksui.NewListViewModel(f.App)), drink.EntityUID()
		}},
		{name: "ingredient", model: func(t testing.TB, f *testutil.Fixture) (views.ViewModel, cedar.EntityUID) {
			ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Tonic", Category: ingredientsmodels.CategoryMixer, Unit: measurement.UnitMl})
			return tuitest.InitAndLoad(t, ingredientsui.NewListViewModel(f.App)), ingredient.EntityUID()
		}},
		{name: "inventory", model: func(t testing.TB, f *testutil.Fixture) (views.ViewModel, cedar.EntityUID) {
			ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
			item := testutil.SetInventory(t, f, inventorymodels.Update{
				IngredientID: ingredient.ID, Amount: measurement.MustAmount(5, ingredient.Unit),
				CostPerUnit: money.NewPriceFromCents(100, currency.USD),
			})
			return tuitest.InitAndLoad(t, inventoryui.NewListViewModel(f.App)), item.EntityUID()
		}},
		{name: "menu", model: func(t testing.TB, f *testutil.Fixture) (views.ViewModel, cedar.EntityUID) {
			menu := testutil.CreateMenu(t, f, "Happy Hour")
			return tuitest.InitAndLoad(t, menusui.NewListViewModel(f.App)), menu.EntityUID()
		}},
		{name: "order", model: func(t testing.TB, f *testutil.Fixture) (views.ViewModel, cedar.EntityUID) {
			ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Rum", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
			drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
				Name: "Rum Sour", Category: drinksmodels.DrinkCategoryCocktail,
				Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, ingredient.Unit)}}, Steps: []string{"Shake"}},
			})
			menu := testutil.CreateMenu(t, f, "Dinner", testutil.WithDrink(drink), testutil.Published())
			order := testutil.PlaceOrder(t, f, ordersmodels.Order{MenuID: menu.ID, Items: []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}}})
			return tuitest.InitAndLoad(t, ordersui.NewListViewModel(f.App)), order.EntityUID()
		}},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			f := testutil.NewFixture(t)
			model, target := scenario.model(t, f)
			model = updateView(t, model, tea.WindowSizeMsg{Width: 120, Height: 40})
			model = updateView(t, model, keyRunes("t"))
			testutil.StringContains(t, model.View(), "Manage tags")
			testutil.StringContains(t, model.View(), "Empty clears all tags")

			model = typeText(t, model, "cedar=deep")
			model = updateViewSettled(t, model, submitKey())

			persisted, err := f.App.Tags.List(f.OwnerContext(), target)
			testutil.Ok(t, err)
			testutil.Equals(t, persisted.Canonical().String(), "cedar=deep")
			testutil.StringContains(t, model.View(), "cedar=deep")
		})
	}
}

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
				testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Tequila", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
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
				ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Tequila", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
				testutil.SetInventory(t, f, inventorymodels.Update{
					IngredientID: ingredient.ID, Amount: measurement.MustAmount(5, ingredient.Unit),
					CostPerUnit: money.NewPriceFromCents(100, currency.USD),
				})
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
				ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Tequila", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
				drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
					Name: "Margarita", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
					Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, ingredient.Unit)}}, Steps: []string{"Shake"}},
				})
				menu := testutil.CreateMenu(t, f, "Dinner", testutil.WithDrink(drink), testutil.Published())
				testutil.PlaceOrder(t, f, ordersmodels.Order{
					MenuID: menu.ID,
					Items:  []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
				})
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

func TestDashboard_NavigatesToTagsShowsHelpAndBack(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	app := NewApp(f.App)

	app = updateAppAndRunCmds(t, app, keyRunes("7"))
	testutil.Equals(t, app.currentView, ViewTags)
	testutil.ErrorIf(t, app.currentViewModel().View() == "", "expected tags workspace")

	app = updateAppAndRunCmds(t, app, keyRunes("?"))
	testutil.IsTrue(t, app.showHelp)
	testutil.ErrorIf(t, app.helpHeight() == 0, "expected expanded tags help")

	app = updateAppOnce(t, app, tea.KeyMsg{Type: tea.KeyEsc})
	testutil.Equals(t, app.currentView, ViewDashboard)
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

func typeText(t testing.TB, model views.ViewModel, value string) views.ViewModel {
	t.Helper()
	for _, r := range value {
		model, _ = model.Update(keyRunes(string(r)))
	}
	return model
}

func updateViewSettled(t testing.TB, model views.ViewModel, msg tea.Msg) views.ViewModel {
	t.Helper()
	updated, cmd := model.Update(msg)
	for _, next := range tuitest.RunCmds(cmd) {
		switch next.(type) {
		case cursor.BlinkMsg, spinner.TickMsg:
			continue
		}
		updated = updateViewSettled(t, updated, next)
	}
	return updated
}

func submitKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyCtrlS}
}
