package testutil_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestFixture_CreateDrinkRoundTrip(t *testing.T) {
	t.Parallel()

	fix := testutil.NewFixture(t)
	tequila := testutil.CreateIngredient(t, fix, ingredientsmodels.Ingredient{Name: "Tequila", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	lime := testutil.CreateIngredient(t, fix, ingredientsmodels.Ingredient{Name: "Lime Juice", Category: ingredientsmodels.CategoryJuice, Unit: measurement.UnitOz})
	created := testutil.CreateDrink(t, fix, drinksmodels.Drink{
		Name: "Margarita", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{IngredientID: tequila.ID, Amount: measurement.MustAmount(2, tequila.Unit)},
				{IngredientID: lime.ID, Amount: measurement.MustAmount(1, lime.Unit)},
			},
			Steps: []string{"Shake"},
		},
	})

	res, err := fix.Drinks.Get(fix.OwnerContext(), created.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, res.Name != "Margarita", "expected Margarita, got %q", res.Name)
}

func TestFixture_BuildsDrinkFromIngredientAndAddsItToMenu(t *testing.T) {
	t.Parallel()

	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Fresh Lime", Category: ingredientsmodels.CategoryJuice, Unit: measurement.UnitOz})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "Daiquiri", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1.5, ingredient.Unit)}}, Steps: []string{"Shake"}},
	})
	menu := testutil.CreateMenu(t, f, "Classics", testutil.WithDrink(drink))

	testutil.Equals(t, drink.Recipe.Ingredients[0].IngredientID, ingredient.ID)
	testutil.Equals(t, drink.Recipe.Ingredients[0].Amount, measurement.MustAmount(1.5, measurement.UnitOz))
	testutil.Equals(t, menu.Items[0].DrinkID, drink.ID)
}

func TestFixture_IsolatedParallelStores(t *testing.T) {
	t.Parallel()
	t.Run("A", func(t *testing.T) {
		t.Parallel()
		fix := testutil.NewFixture(t)
		gin := testutil.CreateIngredient(t, fix, ingredientsmodels.Ingredient{Name: "Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
		testutil.CreateDrink(t, fix, drinksmodels.Drink{
			Name: "A", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
			Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: gin.ID, Amount: measurement.MustAmount(2, gin.Unit)}}, Steps: []string{"Mix"}},
		})
		res, err := fix.Drinks.List(fix.OwnerContext(), drinks.ListRequest{})
		testutil.Ok(t, err)
		testutil.ErrorIf(t, len(res.Items) != 1, "expected 1 drink, got %d", len(res.Items))
	})

	t.Run("B", func(t *testing.T) {
		t.Parallel()
		fix := testutil.NewFixture(t)
		vodka := testutil.CreateIngredient(t, fix, ingredientsmodels.Ingredient{Name: "Vodka", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
		testutil.CreateDrink(t, fix, drinksmodels.Drink{
			Name: "B", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
			Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: vodka.ID, Amount: measurement.MustAmount(2, vodka.Unit)}}, Steps: []string{"Mix"}},
		})
		res, err := fix.Drinks.List(fix.OwnerContext(), drinks.ListRequest{})
		testutil.Ok(t, err)
		testutil.ErrorIf(t, len(res.Items) != 1, "expected 1 drink, got %d", len(res.Items))
	})
}

func TestFixture_RecordsMetrics(t *testing.T) {
	t.Parallel()

	fix := testutil.NewFixture(t)

	_, err := fix.Drinks.List(fix.OwnerContext(), drinks.ListRequest{})
	testutil.Ok(t, err)

	got := fix.Metrics.CounterValue(telemetry.MetricQueryTotal, "Drink.list", "success")
	testutil.ErrorIf(t, got < 1, "expected query metric increment, got %v", got)
}
