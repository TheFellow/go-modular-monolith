package testutil_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
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

func TestFixture_IsolatedParallelStores(t *testing.T) {
	t.Run("A", func(t *testing.T) {
		t.Parallel()
		fix := testutil.NewFixture(t)
		fix.Bootstrap().WithBasicIngredients()
		fix.CreateDrink("A").With("Gin", 2.0).Build()
		res, err := fix.Drinks.List(fix.OwnerContext(), drinks.ListRequest{})
		testutil.Ok(t, err)
		testutil.ErrorIf(t, len(res) != 1, "expected 1 drink, got %d", len(res))
	})

	t.Run("B", func(t *testing.T) {
		t.Parallel()
		fix := testutil.NewFixture(t)
		fix.Bootstrap().WithBasicIngredients()
		fix.CreateDrink("B").With("Vodka", 2.0).Build()
		res, err := fix.Drinks.List(fix.OwnerContext(), drinks.ListRequest{})
		testutil.Ok(t, err)
		testutil.ErrorIf(t, len(res) != 1, "expected 1 drink, got %d", len(res))
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
