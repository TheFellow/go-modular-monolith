package tui

import (
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	tea "github.com/charmbracelet/bubbletea"
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
)

func TestRetireIngredientVMAppliesExplicitReplacement(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	replacement := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Replacement", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	retired := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Retired", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{Name: "TUI replacement", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe, Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: retired.ID, Amount: measurement.MustAmount(1, retired.Unit)}}, Steps: []string{"Mix"}}})
	vm := NewRetireIngredientVM(f.App, retired)
	testutil.Ok(t, vm.replacement.SetValue(replacement.ID.String()))
	testutil.Ok(t, vm.ratio.SetValue("1"))
	msg := vm.submit()()
	if failure, ok := msg.(DeleteErrorMsg); ok {
		testutil.Fail(t, "retirement failed: %v", failure.Err)
	}
	got, err := f.Drinks.Get(f.OwnerContext(), drink.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got.Recipe.Ingredients[0].IngredientID, replacement.ID)
}

func TestRetireIngredientVMWithdrawsStockWithReason(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Withdrawn", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, inventorymodels.Update{CostPerUnit: money.NewPriceFromCents(100, currency.USD), IngredientID: ingredient.ID, Amount: measurement.MustAmount(10, ingredient.Unit)})
	vm := NewRetireIngredientVM(f.App, ingredient)
	testutil.Ok(t, vm.withdraw.SetValue(true))
	testutil.Ok(t, vm.reason.SetValue("Contamination"))
	_, cmd := vm.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	result := cmd()
	if failure, ok := result.(DeleteErrorMsg); ok {
		testutil.Fail(t, "retirement failed: %v", failure.Err)
	}
	stock, err := f.Inventory.Get(f.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Status, inventorymodels.StatusQuarantined)
	testutil.Equals(t, stock.Reason, "Contamination")
}

func TestRetirementFormKeepsLabelsAndHelpVisibleWhenResized(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Retirement display", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	program := &ingredientsPagingProgram{vm: NewListViewModel(f.App)}
	driver := tuitest.NewDriver(t, program)
	driver.Resize(120, 40)
	driver.Press("R")
	driver.RequireText("Replacement ID (optional)", "Existing stock", "Reason", "ctrl+s retire · esc back")
	driver.Resize(100, 30)
	driver.RequireText("Replacement ID (optional)", "Existing stock", "Reason", "ctrl+s retire · esc back")
	driver.Resize(80, 18)
	driver.Press("down")
	driver.Press("down")
	driver.Press("down")
	driver.RequireText("Reason", "ctrl+s retire · esc back")
	driver.RequireViewport(80, 18)
}
