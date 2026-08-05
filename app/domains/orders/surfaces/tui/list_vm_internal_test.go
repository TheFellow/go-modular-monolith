//nolint:paralleltest // terminal program lifecycles intentionally run serially.
package tui

import (
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"strings"
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/dialog"
	tea "github.com/charmbracelet/bubbletea"
)

func TestListVMRejectsDuplicateMutationWhileCommandIsInFlight(t *testing.T) {
	t.Parallel()
	vm := NewListViewModel(nil)
	vm.setSize(100, 30)
	vm.mode = listModeConfirmingComplete
	vm.completeTarget = &models.Order{ID: entity.NewOrderID()}

	_, command := vm.Update(dialog.ConfirmMsg{})
	testutil.ErrorIf(t, command == nil, "confirmation did not create completion command")
	testutil.ErrorIf(t, !vm.mutating, "completion did not claim mutation ownership")
	testutil.ErrorIf(t, !vm.Interaction().HandlesBack, "back navigation was not captured during mutation")
	testutil.ErrorIf(t, !vm.Interaction().CapturesText, "global navigation was not captured during mutation")
	testutil.ErrorIf(t, !strings.Contains(vm.View(), "Updating order"), "mutation progress is not visible")

	_, duplicate := vm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	testutil.ErrorIf(t, duplicate != nil, "duplicate mutation produced a command")
	testutil.ErrorIf(t, !vm.mutating, "duplicate input released mutation ownership")

	vm.Update(CompleteErrorMsg{Err: errors.New("test")})
	testutil.ErrorIf(t, vm.mutating, "failed completion did not release mutation ownership")
}

func TestListVMPagesForwardAndBackAcrossMoreThanOneServerPage(t *testing.T) {
	fix := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, fix, ingredientsmodels.Ingredient{Name: "Paging Base", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.SetInventory(t, fix, inventorymodels.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(500, measurement.UnitOz), CostPerUnit: money.NewPriceFromCents(10, currency.USD)})
	drink := testutil.CreateDrink(t, fix, drinksmodels.Drink{Name: "Paging Drink", Category: drinksmodels.DrinkCategoryHighball, Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}}, Steps: []string{"Build"}}})
	menu := testutil.CreateMenu(t, fix, "Paging Menu", testutil.WithDrink(drink), testutil.Published())
	for range 101 {
		testutil.PlaceOrder(t, fix, models.Order{MenuID: menu.ID, Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 1}}})
	}

	vm := NewListViewModel(fix.App)
	vm.setSize(100, 30)
	first := vm.loadOrders()().(OrdersLoadedMsg)
	vm.Update(first)
	testutil.Equals(t, len(vm.list.Items()), 100)
	testutil.ErrorIf(t, vm.next == "", "first server page omitted next cursor")

	_, nextCommand := vm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	testutil.ErrorIf(t, nextCommand == nil, "forward paging produced no command")
	vm.Update(vm.loadOrders()().(OrdersLoadedMsg))
	testutil.Equals(t, len(vm.list.Items()), 1)
	testutil.Equals(t, len(vm.history), 1)

	_, previousCommand := vm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	testutil.ErrorIf(t, previousCommand == nil, "back paging produced no command")
	vm.Update(vm.loadOrders()().(OrdersLoadedMsg))
	testutil.Equals(t, len(vm.list.Items()), 100)
	testutil.Equals(t, len(vm.history), 0)

	vm.request.Filter = `status ==`
	failed := vm.loadOrders()().(OrdersLoadedMsg)
	testutil.ErrorIf(t, failed.Err == nil, "invalid server filter did not fail")
	vm.Update(failed)
	testutil.Equals(t, len(vm.list.Items()), 100)
	testutil.ErrorIf(t, vm.err == nil, "invalid filter error was not presented")
}
