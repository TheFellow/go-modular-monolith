package testutil_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestFixture_CreateDrinkRoundTrip(t *testing.T) {
	t.Parallel()

	fix := testutil.NewFixture(t)
	fix.Bootstrap().WithBasicIngredients()

	created := fix.CreateDrink("Margarita").
		With("Tequila", 2.0).
		With("Lime Juice", 1.0).
		Build()

	res, err := fix.Drinks.Get(fix.OwnerContext(), created.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, res.Name != "Margarita", "expected Margarita, got %q", res.Name)
}

func TestFixture_BuildsDrinkFromIngredientAndAddsItToMenu(t *testing.T) {
	t.Parallel()

	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ingredient := b.WithIngredient("Fresh Lime", measurement.UnitOz)
	drink := f.CreateDrink("Daiquiri").WithIngredient(ingredient, 1.5).Build()
	menu := b.AddDrinks(b.WithMenu("Classics"), drink)

	testutil.Equals(t, drink.Recipe.Ingredients[0].IngredientID, ingredient.ID)
	testutil.Equals(t, drink.Recipe.Ingredients[0].Amount, measurement.MustAmount(1.5, measurement.UnitOz), testutil.EquateAmounts(0.000001))
	testutil.Equals(t, menu.Items[0].DrinkID, drink.ID)
}

func TestFixture_IsolatedParallelStores(t *testing.T) {
	t.Parallel()
	t.Run("A", func(t *testing.T) {
		t.Parallel()
		fix := testutil.NewFixture(t)
		fix.Bootstrap().WithBasicIngredients()
		fix.CreateDrink("A").With("Gin", 2.0).Build()
		res, err := fix.Drinks.List(fix.OwnerContext(), drinks.ListRequest{})
		testutil.Ok(t, err)
		testutil.ErrorIf(t, len(res.Items) != 1, "expected 1 drink, got %d", len(res.Items))
	})

	t.Run("B", func(t *testing.T) {
		t.Parallel()
		fix := testutil.NewFixture(t)
		fix.Bootstrap().WithBasicIngredients()
		fix.CreateDrink("B").With("Vodka", 2.0).Build()
		res, err := fix.Drinks.List(fix.OwnerContext(), drinks.ListRequest{})
		testutil.Ok(t, err)
		testutil.ErrorIf(t, len(res.Items) != 1, "expected 1 drink, got %d", len(res.Items))
	})
}

func TestFixture_RecordsMetrics(t *testing.T) {
	t.Parallel()

	fix := testutil.NewFixture(t)
	fix.Bootstrap().WithBasicIngredients()

	_, err := fix.Drinks.List(fix.OwnerContext(), drinks.ListRequest{})
	testutil.Ok(t, err)

	got := fix.Metrics.CounterValue(telemetry.MetricQueryTotal, "Drink.list", "success")
	testutil.ErrorIf(t, got < 1, "expected query metric increment, got %v", got)
}
