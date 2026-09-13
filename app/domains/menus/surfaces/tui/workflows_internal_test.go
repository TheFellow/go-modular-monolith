//nolint:paralleltest // terminal workflow request ordering intentionally runs serially.
package tui

import (
	"fmt"
	"strings"
	"testing"

	drinkmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/queries"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestWorkflowResponsesIgnoreSupersededRequests(t *testing.T) {
	vm := NewListViewModel(nil)
	vm.workflowID = 2
	vm.mode = listModeAnalyzing
	vm.analysis = newAnalysisVM()

	vm.Update(analysisLoadedMsg{workflowID: 1, value: queries.MenuAnalytics{TotalCount: 99}})

	testutil.ErrorIf(t, vm.analysis.result != nil, "%v", "a superseded analysis response replaced the active workflow")
}

func TestAnalysisTextDoesNotPresentUnknownCostAsKnown(t *testing.T) {
	cost := money.NewPriceFromCents(123, currency.USD)
	view := analysisText(queries.MenuAnalytics{Items: []queries.MenuItemAnalytics{{
		Name: "Unknown cost drink", Cost: &cost, CostUnknown: true,
	}}})

	testutil.ErrorIf(t, !strings.Contains(view, "Cost: unknown"), "expected unknown cost marker, got:\n%s", view)
	testutil.ErrorIf(t, strings.Contains(view, "$1.23"), "unknown cost leaked a misleading amount:\n%s", view)
}

func TestAnalysisShowsEveryAppliedSubstitution(t *testing.T) {
	f := testutil.NewFixture(t)
	recipe := drinkmodels.Recipe{Steps: []string{"Stir"}}
	var expected []string
	for i := range 2 {
		original := testutil.CreateIngredient(t, f, ingredientmodels.Ingredient{Name: fmt.Sprintf("Original %d", i), Category: ingredientmodels.CategorySpirit, Unit: measurement.UnitOz})
		substitute := testutil.CreateIngredient(t, f, ingredientmodels.Ingredient{Name: fmt.Sprintf("Substitute %d", i), Category: ingredientmodels.CategorySpirit, Unit: measurement.UnitOz})
		ratio := float64(i + 1)
		_, err := f.Ingredients.SetSubstitution(f.OwnerContext(), &ingredientmodels.SubstitutionRule{IngredientID: original.ID, SubstituteID: substitute.ID, Ratio: ratio, QualityImpact: ingredientmodels.QualitySimilar})
		testutil.Ok(t, err)
		testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: substitute.ID, Amount: measurement.MustAmount(20, measurement.UnitOz), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
		recipe.Ingredients = append(recipe.Ingredients, drinkmodels.RecipeIngredient{IngredientID: original.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)})
		expected = append(expected, fmt.Sprintf("sub: %s for %s; ratio %g; quality similar", substitute.ID.String(), original.ID.String(), ratio))
	}
	drink := testutil.CreateDrink(t, f, drinkmodels.Drink{Name: "Two substitutions", Category: drinkmodels.DrinkCategoryCocktail, Recipe: recipe})
	menu := testutil.CreateMenu(t, f, "Substitution analysis", testutil.WithDrink(drink))
	analysis, err := f.Menus.Analyze(f.OwnerContext(), *menu, 0.75)
	testutil.Ok(t, err)
	testutil.Equals(t, len(analysis.Items[0].Substitutions), 2)
	for _, want := range expected {
		testutil.StringContains(t, analysisText(analysis), want)
	}
	testutil.StringContains(t, analysisText(analysis), drink.ID.String())
	analysis.Items[0].CostUnknown = true
	testutil.StringContains(t, analysisText(analysis), "Cost: unknown")
}
