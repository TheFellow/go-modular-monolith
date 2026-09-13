//nolint:paralleltest // deterministic keyboard workflows own mutable view models.
package tui

import (
	"testing"

	application "github.com/TheFellow/go-modular-monolith/app"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	tea "github.com/charmbracelet/bubbletea"
)

type amendmentProgram struct{ vm *ListViewModel }

func (p *amendmentProgram) Init() tea.Cmd { return p.vm.Init() }
func (p *amendmentProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := p.vm.Update(msg)
	return p, cmd
}
func (p *amendmentProgram) View() string { return p.vm.View() }
func tuiAmendmentFixture(t *testing.T) (*testutil.Fixture, *models.Order, entity.IngredientID) {
	t.Helper()
	f := testutil.NewFixture(t)
	lime := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Lime", Category: ingredientsmodels.CategoryJuice, Unit: measurement.UnitOz})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{Name: "Cooler", Category: drinksmodels.DrinkCategoryMocktail, Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: lime.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}}, Steps: []string{"Shake | gently"}}})
	menu := testutil.CreateMenu(t, f, "Accepted menu", testutil.WithDrink(drink), testutil.Published())
	replacement := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Lemon", Category: ingredientsmodels.CategoryJuice, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: replacement.ID, Amount: measurement.MustAmount(100, measurement.UnitOz), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	order := testutil.PlaceOrder(t, f, models.Order{MenuID: menu.ID, Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 2}}})
	return f, order, replacement.ID
}
func enterAmendment(driver *tuitest.Driver, replacement entity.IngredientID) {
	driver.Press("a")
	driver.Press("Guest approved")
	driver.Send(tea.KeyMsg{Type: tea.KeyTab})
	driver.Press(replacement.String())
	driver.Send(tea.KeyMsg{Type: tea.KeyTab})
}
func TestAmendmentKeyboardPreservesAcceptanceAndLiteralSteps(t *testing.T) {
	f, order, replacement := tuiAmendmentFixture(t)
	vm := NewListViewModel(f.App)
	driver := tuitest.NewDriver(t, &amendmentProgram{vm: vm})
	driver.Resize(120, 35)
	enterAmendment(driver, replacement)
	driver.RequireText("Amend order", "revision", "quantity ratio")
	driver.Press("ctrl+s")
	stored, err := f.Orders.Get(f.OwnerContext(), order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.Acceptance, order.Acceptance)
	testutil.Equals(t, stored.Plan[0].Ingredients[0].IngredientID, replacement)
	testutil.Equals(t, stored.Plan[0].Steps, []string{"Shake | gently"})
	testutil.Equals(t, len(stored.Amendments), 1)
}
func TestAmendmentBatchKeyboardRetainsQueueOnConflict(t *testing.T) {
	f, order, replacement := tuiAmendmentFixture(t)
	second := testutil.PlaceOrder(t, f, models.Order{MenuID: order.MenuID, Items: order.Items})
	vm := NewListViewModel(f.App)
	driver := tuitest.NewDriver(t, &amendmentProgram{vm: vm})
	driver.Resize(120, 35)
	enterAmendment(driver, replacement)
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlB})
	driver.Press("down")
	enterAmendment(driver, replacement)
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlB})
	testutil.Equals(t, len(vm.amendmentQueue), 2)
	testutil.Equals(t, vm.Interaction().HandlesBack, true)
	_, err := f.Orders.Cancel(f.OwnerContext(), second)
	testutil.Ok(t, err)
	driver.Press("esc")
	testutil.Equals(t, vm.mode, listModeAmendmentBatch)
	driver.RequireText("Replace", replacement.String(), "ratio 1")
	driver.Press("ctrl+s")
	testutil.ErrorIsConflict(t, vm.err)
	testutil.Equals(t, len(vm.amendmentQueue), 2)
	unchanged, err := f.Orders.Get(f.OwnerContext(), order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, unchanged.Revision, order.Revision)
	testutil.Equals(t, unchanged.Plan, order.Plan)
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlX})
	testutil.Equals(t, len(vm.amendmentQueue), 0)
}
func TestAmendmentKeyboardIsHiddenWithoutPermission(t *testing.T) {
	f, _, _ := tuiAmendmentFixture(t)
	vm := NewListViewModel(application.NewSession(f.ActorContext("sommelier"), f.App.App))
	driver := tuitest.NewDriver(t, &amendmentProgram{vm: vm})
	driver.Press("a")
	testutil.Equals(t, vm.mode, listModeBrowsing)
	for _, key := range vm.ShortHelp() {
		testutil.ErrorIf(t, key.Help().Key == "a", "amend key exposed without permission")
	}
}
func TestCancellationKeyboardCapturesReason(t *testing.T) {
	f, order, _ := tuiAmendmentFixture(t)
	vm := NewListViewModel(f.App)
	driver := tuitest.NewDriver(t, &amendmentProgram{vm: vm})
	driver.Resize(120, 35)
	driver.Press("x")
	driver.Press("Guest left")
	driver.Press("ctrl+s")
	driver.Press("ctrl+s") // retain tags and continue to confirmation
	// Dangerous confirmations start on Cancel; move to Confirm before accepting.
	driver.Send(tea.KeyMsg{Type: tea.KeyTab})
	driver.Press("enter")
	stored, err := f.Orders.Get(f.OwnerContext(), order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.CancellationReason, "Guest left")
	testutil.Equals(t, stored.Status, models.OrderStatusCancelled)
}
func TestLoadedOrdersUseAcceptedMenuNameWithoutLiveLookup(t *testing.T) {
	f := testutil.NewFixture(t)
	vm := NewListViewModel(f.App)
	vm.Update(OrdersLoadedMsg{Orders: []models.Order{{ID: entity.NewOrderID(), MenuID: entity.NewMenuID(), Status: models.OrderStatusCompleted, Acceptance: models.AcceptanceSnapshot{MenuName: "Deleted accepted menu"}}}})
	testutil.Ok(t, vm.err)
	testutil.StringContains(t, vm.View(), "Deleted accepted menu")
}
