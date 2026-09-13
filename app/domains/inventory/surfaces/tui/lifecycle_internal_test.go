//nolint:paralleltest // keyboard workflows intentionally run serially.
package tui

import (
	"testing"

	application "github.com/TheFellow/go-modular-monolith/app"
	ingredientmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	tea "github.com/charmbracelet/bubbletea"
)

type inventoryProgram struct{ *ListViewModel }

func (m inventoryProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.ListViewModel.Update(msg)
	return m, cmd
}

func TestInventoryLifecycleKeyboard(t *testing.T) {
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientmodels.Ingredient{Name: "Lifecycle stock", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, models.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(10, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	vm := NewListViewModel(f.App)
	driver := tuitest.NewDriver(t, inventoryProgram{vm})
	driver.Resize(120, 30)
	driver.Press("x")
	driver.RequireText("Quarantine stock")
	driver.Press("enter")
	driver.Press("Quality review")
	driver.Press("enter")
	driver.Press("ctrl+s")
	stock, err := f.Inventory.Get(f.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Status, models.StatusQuarantined)
	driver.Press("u")
	driver.Press("enter")
	driver.Press("Approved")
	driver.Press("enter")
	driver.Press("ctrl+s")
	stock, err = f.Inventory.Get(f.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Status, models.StatusActive)
	driver.Press("x")
	driver.Press("enter")
	driver.Press("Expired")
	driver.Press("enter")
	driver.Press("ctrl+s")
	driver.Press("d")
	driver.Press("enter")
	driver.Press("0.0001")
	driver.Press("enter")
	driver.Press("down")
	driver.Press("enter")
	driver.Press("Sample discarded")
	driver.Press("enter")
	driver.Press("ctrl+s")
	stock, err = f.Inventory.Get(f.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Amount, measurement.MustAmount(9.9999, measurement.UnitOz))
	driver.Press("h")
	driver.RequireText("Stock movement history")
	testutil.Equals(t, len(vm.movements), 5)
	driver.Press("esc")
	driver.Press("d")
	driver.Resize(80, 15)
	driver.RequireViewport(80, 15)
}

func TestInventoryLifecycleRejectsStaleAndDeniedEditors(t *testing.T) {
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientmodels.Ingredient{Name: "Stale stock", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	stock := testutil.SetInventory(t, f, models.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(10, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	vm := newLifecycleVM(f.App, InventoryRow{Ingredient: *ingredient, Inventory: *stock}, inventory.ControlQuarantine)
	testutil.Ok(t, vm.reason.SetValue("Stale editor"))
	_, err := f.Inventory.Disposition(f.OwnerContext(), models.Disposition{IngredientID: ingredient.ID, Revision: stock.Revision, Quarantine: true, Reason: "Other editor"})
	testutil.Ok(t, err)
	result := vm.submit()()
	failure := testutil.Cast[lifecycleErrorMsg](t, result)
	testutil.ErrorIsConflict(t, failure.Err)
	denied := application.NewSession(f.ActorContext("bartender"), f.App.App)
	list := NewListViewModel(denied)
	driver := tuitest.NewDriver(t, inventoryProgram{list})
	driver.Press("u")
	testutil.Equals(t, list.mode, listModeBrowsing)
	testutil.Equals(t, list.actions[inventory.ControlRelease].Visible, false)
}

func TestInventorySetPreservesConvertedQuantityAndEditableCostUnit(t *testing.T) {
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientmodels.Ingredient{Name: "Converted stock", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, models.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(12.5, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(325, currency.USD)})
	ingredient.Unit = measurement.UnitMl
	ingredient, err := f.Ingredients.Update(f.OwnerContext(), ingredient)
	testutil.Ok(t, err)
	stock, err := f.Inventory.Get(f.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	vm := NewSetInventoryVM(f.App, InventoryRow{Ingredient: *ingredient, Inventory: *stock})
	testutil.Ok(t, vm.cost.SetValue("$4.00"))
	saved := testutil.Cast[InventorySetMsg](t, vm.submit()())
	testutil.Equals(t, saved.Inventory.Amount, stock.Amount)
	testutil.Equals(t, saved.Inventory.CostUnit, measurement.UnitOz)
	vm = NewSetInventoryVM(f.App, InventoryRow{Ingredient: *ingredient, Inventory: *saved.Inventory})
	testutil.Ok(t, vm.cost.SetValue("$0.25"))
	testutil.Ok(t, vm.costUnit.SetValue(measurement.UnitMl))
	saved = testutil.Cast[InventorySetMsg](t, vm.submit()())
	testutil.Equals(t, saved.Inventory.CostUnit, measurement.UnitMl)
}

func TestReceiveNewInventoryKeyboardAndConcurrentReceipt(t *testing.T) {
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientmodels.Ingredient{Name: "New stock", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitMl})
	vm := NewListViewModel(f.App)
	driver := tuitest.NewDriver(t, inventoryProgram{vm})
	driver.Resize(120, 30)
	driver.Press("c")
	driver.RequireText("Receive new stock")
	driver.Press("ctrl+s")
	driver.RequireText("Set Inventory: New stock")
	driver.Press("enter")
	driver.Press("ctrl+u")
	driver.Press("0.12345")
	driver.Press("enter")
	driver.Press("ctrl+s")
	stock, err := f.Inventory.Get(f.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Amount, measurement.MustAmount(0.12345, measurement.UnitMl))
	driver.Press("c")
	testutil.Equals(t, len(vm.stocking.candidates), 0)
	driver.Press("esc")
	another := testutil.CreateIngredient(t, f, ingredientmodels.Ingredient{Name: "Concurrent stock", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitMl})
	driver.Press("c")
	driver.Press("ctrl+s")
	testutil.SetInventory(t, f, models.Update{IngredientID: another.ID, Amount: measurement.MustAmount(5, another.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	driver.Press("enter")
	driver.Press("ctrl+u")
	driver.Press("2")
	driver.Press("enter")
	driver.Press("ctrl+s")
	testutil.Equals(t, vm.mode, listModeSetting)
	testutil.ErrorIsConflict(t, vm.set.err)
	stock, err = f.Inventory.Get(f.OwnerContext(), another.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Amount.Value(), 5.0)
}
