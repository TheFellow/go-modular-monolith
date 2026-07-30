package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
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
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/app/surfaces/tui/components"
	"github.com/TheFellow/go-modular-monolith/app/surfaces/tui/styles"
	"github.com/TheFellow/go-modular-monolith/app/surfaces/tui/views"
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

type tagViewScenario struct {
	name, nav, title string
	model            func(testing.TB, *testutil.Fixture) (views.ViewModel, cedar.EntityUID)
}

func TestEveryTopLevelViewRendersValidFramesAcrossViewportSizes(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	driver := tuitest.NewDriver(t, NewApp(f.App))
	driver.Resize(MinWidth, MinHeight)

	for _, scenario := range []struct {
		nav, title string
	}{
		{nav: "1", title: "Mixology > Drinks"},
		{nav: "2", title: "Mixology > Ingredients"},
		{nav: "3", title: "Mixology > Inventory"},
		{nav: "4", title: "Mixology > Menus"},
		{nav: "5", title: "Mixology > Orders"},
		{nav: "6", title: "Mixology > Audit"},
		{nav: "7", title: "Mixology > Tags"},
	} {
		driver.Press(scenario.nav)
		driver.RequireText(scenario.title)
		driver.Resize(120, 40)
		driver.Resize(MinWidth, MinHeight)
		driver.Press("esc")
		driver.RequireText("Mixology > Dashboard")
	}
}

func tagViewScenarios() []tagViewScenario {
	return []tagViewScenario{
		{name: "drink", nav: "1", title: "Drinks", model: func(t testing.TB, f *testutil.Fixture) (views.ViewModel, cedar.EntityUID) {
			ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Lime", Category: ingredientsmodels.CategoryJuice, Unit: measurement.UnitOz})
			drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
				Name: "Daiquiri", Category: drinksmodels.DrinkCategoryCocktail,
				Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, ingredient.Unit)}}, Steps: []string{"Shake"}},
			})
			return tuitest.InitAndLoad(t, drinksui.NewListViewModel(f.App)), drink.EntityUID()
		}},
		{name: "ingredient", nav: "2", title: "Ingredients", model: func(t testing.TB, f *testutil.Fixture) (views.ViewModel, cedar.EntityUID) {
			ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Tonic", Category: ingredientsmodels.CategoryMixer, Unit: measurement.UnitMl})
			return tuitest.InitAndLoad(t, ingredientsui.NewListViewModel(f.App)), ingredient.EntityUID()
		}},
		{name: "inventory", nav: "3", title: "Inventory", model: func(t testing.TB, f *testutil.Fixture) (views.ViewModel, cedar.EntityUID) {
			ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
			item := testutil.SetInventory(t, f, inventorymodels.Update{
				IngredientID: ingredient.ID, Amount: measurement.MustAmount(5, ingredient.Unit),
				CostPerUnit: money.NewPriceFromCents(100, currency.USD),
			})
			return tuitest.InitAndLoad(t, inventoryui.NewListViewModel(f.App)), item.EntityUID()
		}},
		{name: "menu", nav: "4", title: "Menus", model: func(t testing.TB, f *testutil.Fixture) (views.ViewModel, cedar.EntityUID) {
			menu := testutil.CreateMenu(t, f, "Happy Hour")
			return tuitest.InitAndLoad(t, menusui.NewListViewModel(f.App)), menu.EntityUID()
		}},
		{name: "order", nav: "5", title: "Orders", model: func(t testing.TB, f *testutil.Fixture) (views.ViewModel, cedar.EntityUID) {
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
}

func TestTagHotkeyManagesEveryOperationalEntity(t *testing.T) {
	t.Parallel()

	for _, scenario := range tagViewScenarios() {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			f := testutil.NewFixture(t)
			model, target := scenario.model(t, f)
			model = updateView(t, model, tea.WindowSizeMsg{Width: 120, Height: 40})
			model = updateView(t, model, keyRunes("t"))
			testutil.StringContains(t, model.View(), "Manage tags")
			testutil.StringContains(t, model.View(), "Empty clears all tags")
			model = updateView(t, model, components.TagsSavedMsg{Target: entity.NewAuditEntryID().EntityUID()})
			testutil.StringContains(t, model.View(), "Manage tags")

			model = typeText(t, model, "cedar=deep")
			model = updateViewSettled(t, model, submitKey())

			persisted, err := f.App.Tags.List(f.OwnerContext(), target)
			testutil.Ok(t, err)
			testutil.Equals(t, persisted.Canonical().String(), "cedar=deep")
			testutil.StringContains(t, model.View(), "cedar=deep")
		})
	}
}

func TestDomainModalOwnsKeysThatAreBrowsingShortcuts(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name, activate, collision, expected string
		setup                               tagViewScenario
	}{
		{name: "drink create", activate: "c", collision: "t", expected: "Name", setup: tagViewScenarios()[0]},
		{name: "ingredient create", activate: "c", collision: "t", expected: "Name", setup: tagViewScenarios()[1]},
		{name: "inventory adjust", activate: "a", collision: "t", expected: "Adjust: Gin", setup: tagViewScenarios()[2]},
		{name: "menu create", activate: "c", collision: "t", expected: "Name", setup: tagViewScenarios()[3]},
		{name: "order cancel", activate: "x", collision: "t", expected: "Complete tags (optional)", setup: tagViewScenarios()[4]},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			f := testutil.NewFixture(t)
			model, _ := scenario.setup.model(t, f)
			model = updateView(t, model, tea.WindowSizeMsg{Width: 120, Height: 40})
			model = updateViewSettled(t, model, keyRunes(scenario.activate))
			testutil.StringContains(t, model.View(), scenario.expected)

			model = updateViewSettled(t, model, keyRunes(scenario.collision))
			testutil.StringContains(t, model.View(), scenario.expected)
			testutil.ErrorIf(t, strings.Contains(model.View(), "Manage tags"), "browsing shortcut escaped the active modal:\n%s", model.View())
		})
	}
}

func TestE2E_ListFilterOwnsPrintableShortcutsAndEscape(t *testing.T) {
	t.Parallel()

	scenarios := tagViewScenarios()
	scenarios = append(scenarios, tagViewScenario{
		name: "audit", nav: "6", title: "Audit",
		model: func(t testing.TB, f *testutil.Fixture) (views.ViewModel, cedar.EntityUID) {
			ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
				Name: "Audited Tonic", Category: ingredientsmodels.CategoryMixer, Unit: measurement.UnitMl,
			})
			return tuitest.InitAndLoad(t, auditui.NewListViewModel(f.App)), ingredient.EntityUID()
		},
	})
	for _, scenario := range scenarios {
		if scenario.name == "inventory" {
			continue // Inventory uses a table and intentionally has no text filter.
		}
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			f := testutil.NewFixture(t)
			_, _ = scenario.model(t, f)
			driver := tuitest.NewDriver(t, NewApp(f.App))
			driver.Resize(120, 40)
			driver.Press(scenario.nav)
			driver.Press("f")
			filterTitle := "Filter " + scenario.title
			if scenario.name == "audit" {
				filterTitle = "Query Audit"
			}
			driver.RequireText(filterTitle)
			pressText(driver, "query")
			driver.RequireRunning()
			driver.RequireText(filterTitle)

			driver.Press("esc")
			driver.RequireText("Mixology > " + scenario.title)
			driver.RequireNoText("Mixology > Dashboard")
		})
	}
}

func TestE2E_TagPickerFilterOwnsPrintableShortcutsAndEscape(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	driver := tuitest.NewDriver(t, NewApp(f.App))
	driver.Resize(120, 40)
	driver.Press("7")
	driver.Press("enter")
	driver.RequireText("Select entity type")
	driver.Press("/")
	pressText(driver, "query")
	driver.RequireRunning()
	driver.RequireText("Filter: query")

	driver.Press("esc")
	driver.RequireText("Select entity type")
	driver.RequireNoText("Inspect entity tags")
}

func TestE2E_TagEditorSupportsReplaceClearCancelAndValidationAcrossEveryEntityView(t *testing.T) {
	t.Parallel()

	for _, scenario := range tagViewScenarios() {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			f := testutil.NewFixture(t)
			_, target := scenario.model(t, f)
			driver := tuitest.NewDriver(t, NewApp(f.App))
			driver.Resize(120, 40)
			driver.Press(scenario.nav)
			driver.RequireText("Mixology > " + scenario.title)
			driver.Press("?")
			driver.RequireText("t manage tags")
			driver.Press("?")

			driver.Press("t")
			driver.RequireText("Manage tags", "Empty clears all tags")
			pressText(driver, "quality=quirky?")
			driver.RequireRunning()
			driver.Press("ctrl+s")
			driver.RequireText("Mixology > "+scenario.title, "quality=quirky?")
			driver.RequireViewport(120, 40)
			requirePersistedTags(t, f, target, "quality=quirky?")

			driver.Press("t")
			driver.RequireText("quality=quirky?")
			driver.Press("ctrl+u")
			driver.Press("ctrl+s")
			driver.RequireText("Tags: (none)")
			requirePersistedTags(t, f, target, "")

			driver.Press("t")
			pressText(driver, "transient")
			driver.Press("esc")
			driver.RequireText("Mixology > "+scenario.title, "Tags: (none)")
			requirePersistedTags(t, f, target, "")

			driver.Press("t")
			pressText(driver, "region=east,region=west")
			driver.Press("ctrl+s")
			driver.RequireText("Manage tags", "invalid tags")
			requirePersistedTags(t, f, target, "")
		})
	}
}

func TestE2E_TagEditorPermissionFailureStaysOpenAndDoesNotPersist(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Protected Tonic", Category: ingredientsmodels.CategoryMixer, Unit: measurement.UnitMl,
	})
	session := app.NewSession(f.ActorContext("bartender"), f.App.App)
	driver := tuitest.NewDriver(t, NewApp(session))
	driver.Resize(100, 40)
	driver.Press("2")
	driver.Press("t")
	pressText(driver, "denied")
	driver.Press("ctrl+s")
	driver.RequireText("Manage tags", "authz denied")
	driver.RequireViewport(100, 40)
	requirePersistedTags(t, f, ingredient.EntityUID(), "")
}

func pressText(driver *tuitest.Driver, value string) {
	for _, r := range value {
		driver.Press(string(r))
	}
}

func requirePersistedTags(t testing.TB, f *testutil.Fixture, target cedar.EntityUID, expected string) {
	t.Helper()
	persisted, err := f.App.Tags.List(f.OwnerContext(), target)
	testutil.Ok(t, err)
	testutil.Equals(t, persisted.Canonical().String(), expected)
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

			testutil.IsTrue(t, model.Interaction().HandlesBack)

			app := NewApp(f.App)
			app.currentView = scenario.view
			app.prevViews = []View{ViewDashboard}
			app.views[scenario.view] = model

			app = updateAppAndRunCmds(t, app, tea.KeyMsg{Type: tea.KeyEsc})
			testutil.Equals(t, app.currentView, scenario.view)

			testutil.IsFalse(t, app.views[scenario.view].Interaction().HandlesBack)

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

func TestBackKey_ClosesExpandedHelpBeforeNavigating(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	driver := tuitest.NewDriver(t, NewApp(f.App))
	driver.Resize(100, 40)
	driver.Press("1")
	driver.Press("?")
	driver.RequireText("t manage tags")
	driver.Press("esc")
	driver.RequireText("Mixology > Drinks")
	driver.RequireNoText("t manage tags")
	driver.Press("esc")
	driver.RequireText("Mixology > Dashboard")
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
	testutil.Equals(t, app.currentView, ViewTags)
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
