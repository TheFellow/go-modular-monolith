//nolint:paralleltest // terminal workflow and picker lifecycles intentionally run serially.
package tui_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus"
	menuauthz "github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	menustui "github.com/TheFellow/go-modular-monolith/app/domains/menus/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
	tea "github.com/charmbracelet/bubbletea"
)

type menuProgram struct{ model tui.ViewModel }

func (m menuProgram) Init() tea.Cmd { return m.model.Init() }
func (m menuProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.model.Update(msg)
	return menuProgram{model: model}, cmd
}
func (m menuProgram) View() string { return m.model.View() }

func newMenuDriver(t *testing.T, f *testutil.Fixture) *tuitest.Driver {
	t.Helper()
	driver := tuitest.NewDriver(t, menuProgram{model: menustui.NewListViewModel(f.App)})
	driver.Resize(140, 45)
	return driver
}

func createMenuTUIDrink(t *testing.T, f *testutil.Fixture, name string) *drinksmodels.Drink {
	t.Helper()
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: name + " Base", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(10, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	return testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: name, Category: drinksmodels.DrinkCategoryCocktail,
		Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}}, Steps: []string{"Build"}},
	})
}

func TestMenuTUIAddsAndRemovesResolvedCommaBearingDrink(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	menu := testutil.CreateMenu(t, f, "Service")
	drink := createMenuTUIDrink(t, f, "Comma, Collins")
	driver := newMenuDriver(t, f)

	driver.Press("a")
	driver.RequireText("Add drink to menu")
	driver.Press("Comma,")
	driver.RequireText("Comma, Collins")
	driver.Send(tea.KeyMsg{Type: tea.KeyTab})
	driver.Press("channel=tui")
	driver.Press("enter")

	got, err := f.App.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(got.Items), 1)
	testutil.Equals(t, got.Items[0].DrinkID, drink.ID)
	testutil.Equals(t, got.Tags.Canonical().String(), "channel=tui")
	testutil.AuditTouches(t, f.LatestAuditEntry(menuauthz.ActionDrinkAdd), menu.EntityUID())
	driver.RequireText("Comma, Collins")

	driver.Press("x")
	driver.RequireText("Remove drink from menu")
	driver.Press("enter")
	driver.RequireText("Remove \"Comma, Collins\"")
	// Dangerous confirmations start on Cancel. Switching focus is required.
	driver.Send(tea.KeyMsg{Type: tea.KeyTab})
	driver.Press("enter")
	got, err = f.App.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(got.Items), 0)
	testutil.AuditTouches(t, f.LatestAuditEntry(menuauthz.ActionDrinkRemove), menu.EntityUID())
}

func TestMenuTUIShowsReadinessBlockerAndDisablesPublish(t *testing.T) {
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Unavailable Base", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{Name: "Unavailable Drink", Category: drinksmodels.DrinkCategoryCocktail, Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, ingredient.Unit)}}, Steps: []string{"Build"}}})
	testutil.CreateMenu(t, f, "Blocked Menu", testutil.WithDrink(drink))
	driver := newMenuDriver(t, f)
	driver.RequireText("Readiness")
	driver.RequireText("blocker")
	driver.RequireText("Resolve menu readiness blockers before")
}

func TestMenuTUIRemoveConfirmationCancelDoesNotMutate(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	drink := createMenuTUIDrink(t, f, "Keep Me")
	menu := testutil.CreateMenu(t, f, "Service", testutil.WithDrink(drink))
	driver := newMenuDriver(t, f)

	driver.Press("x")
	driver.Press("enter")
	driver.Press("enter")
	got, err := f.App.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(got.Items), 1)
}

func TestMenuTUIAnalysisValidatesFiniteRangeAndRendersResults(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	drink := createMenuTUIDrink(t, f, "Costed Collins")
	testutil.CreateMenu(t, f, "Service", testutil.WithDrink(drink))
	driver := newMenuDriver(t, f)

	driver.Press("y")
	driver.RequireText("Menu cost and availability analysis")
	driver.Press("ctrl+u")
	driver.Press("NaN")
	driver.Press("enter")
	driver.RequireText("target margin must be a number between 0 and 1")
	driver.Press("ctrl+u")
	driver.Press("1")
	driver.Press("enter")
	driver.RequireText("target margin must be a number between 0 and 1")
	driver.Press("ctrl+u")
	driver.Press("0.7")
	driver.Press("enter")
	driver.RequireText("Available:")
	driver.RequireText("Costed Collins")
	driver.RequireText("Margin: n/a")
}

func TestMenuTUIPickerBackAndInputOwnership(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	testutil.CreateMenu(t, f, "Service")
	createMenuTUIDrink(t, f, "Searchable")
	driver := newMenuDriver(t, f)

	driver.Press("a")
	driver.Press("q")
	driver.RequireText("No matching drinks")
	testutil.ErrorIf(t, strings.Contains(driver.Screen(), "quit"), "%v", "text input escaped the workflow")
	driver.Press("esc")
	driver.RequireText("Service")
}

func TestMenuTUIAddDrinkAuthorizationFailureDoesNotMutate(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	menu := testutil.CreateMenu(t, f, "Protected")
	createMenuTUIDrink(t, f, "Denied Drink")
	unauthorized := app.NewSession(f.ActorContext("bartender"), f.App.App)
	driver := tuitest.NewDriver(t, menuProgram{model: menustui.NewListViewModel(unauthorized)})
	driver.Resize(140, 45)
	driver.Press("a")
	driver.RequireText("Protected")
	testutil.ErrorIf(t, strings.Contains(driver.Screen(), "Add drink to menu"), "%v", "unauthorized add-drink key opened its workflow")
	got, err := f.App.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(got.Items), 0)
}

func TestMenuTUIPickerTraversesDrinkPages(t *testing.T) {
	f := testutil.NewFixture(t)
	menu := testutil.CreateMenu(t, f, "Large catalog")
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Shared Base", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	var target *drinksmodels.Drink
	for i := range 101 {
		drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
			Name: fmt.Sprintf("Catalog %03d", i), Category: drinksmodels.DrinkCategoryCocktail,
			Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}}, Steps: []string{"Build"}},
		})
		if i == 100 {
			target = drink
		}
	}
	driver := newMenuDriver(t, f)
	driver.Press("a")
	driver.Press("Catalog 100")
	driver.RequireText("Catalog 100")
	driver.Press("enter")
	got, err := f.App.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got.Items[0].DrinkID, target.ID)
}

func TestMenuTUIStateGuardsPublishedCuration(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	drink := createMenuTUIDrink(t, f, "Published Drink")
	menu := testutil.CreateMenu(t, f, "Published", testutil.WithDrink(drink), testutil.Published())
	driver := newMenuDriver(t, f)
	driver.Press("a")
	driver.RequireText("Add drink: Available only while the menu is a")
	driver.Press("x")
	driver.RequireText("Remove drink: Available only while the menu is a")
	got, err := f.App.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(got.Items), 1)
}

func TestMenuTUIFilterOwnsInputSupportsBackAndReportsInvalidExpressions(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	testutil.CreateMenu(t, f, "Visible menu")
	driver := newMenuDriver(t, f)

	driver.Press("f")
	driver.RequireText("Filter Menus")
	driver.Press("q")
	driver.RequireText("Filter Menus")
	driver.Press("esc")
	driver.RequireText("Filter Menus")
	driver.Press("esc")
	driver.RequireText("Visible menu")

	driver.Press("f")
	driver.Press("tab")
	driver.Press("status ==")
	driver.Press("ctrl+s")
	driver.RequireText("invalid")
}

func TestMenuTUIStatusFilterUsesDomainListContract(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	testutil.CreateMenu(t, f, "Draft only")
	drink := createMenuTUIDrink(t, f, "Published drink")
	testutil.CreateMenu(t, f, "Published only", testutil.WithDrink(drink), testutil.Published())
	driver := newMenuDriver(t, f)

	driver.Press("f")
	driver.Press("e")
	driver.Press("down") // all -> draft
	driver.Press("enter")
	driver.Press("ctrl+s")
	driver.RequireText("Draft only")
	testutil.ErrorIf(t, strings.Contains(driver.Screen(), "Published only"), "%v", "published menu escaped the draft status filter")
}

func TestMenuTUITraversesDomainCursorPagesBeyondOneHundred(t *testing.T) {
	f := testutil.NewFixture(t)
	for i := range 101 {
		testutil.CreateMenu(t, f, fmt.Sprintf("Paged menu %03d", i))
	}
	first, err := f.App.Menus.List(f.OwnerContext(), menus.ListRequest{Limit: 100})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, first.Next == "", "%v", "fixture did not produce a second cursor page")
	second, err := f.App.Menus.List(f.OwnerContext(), menus.ListRequest{Cursor: first.Next, Limit: 100})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(second.Items) != 1, "second page contains %d menus", len(second.Items))

	driver := newMenuDriver(t, f)
	driver.Press("]")
	driver.RequireText(second.Items[0].Name)
	driver.Press("[")
	driver.RequireText(first.Items[0].Name)
}

func TestMenuTUIPageSizeControlsCursorWindow(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	testutil.CreateMenu(t, f, "Small page A")
	testutil.CreateMenu(t, f, "Small page B")
	first, err := f.App.Menus.List(f.OwnerContext(), menus.ListRequest{Limit: 1})
	testutil.Ok(t, err)
	second, err := f.App.Menus.List(f.OwnerContext(), menus.ListRequest{Cursor: first.Next, Limit: 1})
	testutil.Ok(t, err)

	driver := newMenuDriver(t, f)
	driver.Press("f")
	driver.Press("tab")
	driver.Press("tab")
	driver.Press("ctrl+u")
	driver.Press("1")
	driver.Press("ctrl+s")
	driver.RequireText(first.Items[0].Name)
	driver.Press("]")
	driver.RequireText(second.Items[0].Name)
}

func TestMenuTUIEditUpdatesDescriptionAndBlankPreservesPublicContract(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	menu := testutil.CreateMenu(t, f, "Described")
	menu.Description = "Original description"
	_, err := f.App.Menus.Update(f.OwnerContext(), menu)
	testutil.Ok(t, err)
	driver := newMenuDriver(t, f)

	driver.Press("e")
	driver.RequireText("Edit Menu")
	driver.Press("tab")
	driver.Press("ctrl+u")
	driver.Press("Updated description")
	driver.Press("ctrl+s")
	got, err := f.App.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got.Description, "Updated description")

	driver.Press("e")
	driver.Press("tab")
	driver.Press("ctrl+u")
	driver.Press("ctrl+s")
	got, err = f.App.Menus.Get(f.OwnerContext(), menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got.Description, "Updated description")
}
