package tui

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
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
		t.Fatalf("retirement failed: %v", failure.Err)
	}
	got, err := f.Drinks.Get(f.OwnerContext(), drink.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got.Recipe.Ingredients[0].IngredientID, replacement.ID)
}
