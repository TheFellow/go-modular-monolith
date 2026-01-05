package testutil_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
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

	res, err := fix.Drinks.Get(fix.Ctx, drinks.GetRequest{ID: created.ID})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, res.Drink.Name != "Margarita", "expected Margarita, got %q", res.Drink.Name)
}

func TestFixture_IsolatedParallelStores(t *testing.T) {
	t.Run("A", func(t *testing.T) {
		t.Parallel()
		fix := testutil.NewFixture(t)
		fix.Bootstrap().WithBasicIngredients()
		fix.CreateDrink("A").With("Gin", 2.0).Build()
		res, err := fix.Drinks.List(fix.Ctx, drinks.ListRequest{})
		testutil.Ok(t, err)
		testutil.ErrorIf(t, len(res.Drinks) != 1, "expected 1 drink, got %d", len(res.Drinks))
	})

	t.Run("B", func(t *testing.T) {
		t.Parallel()
		fix := testutil.NewFixture(t)
		fix.Bootstrap().WithBasicIngredients()
		fix.CreateDrink("B").With("Vodka", 2.0).Build()
		res, err := fix.Drinks.List(fix.Ctx, drinks.ListRequest{})
		testutil.Ok(t, err)
		testutil.ErrorIf(t, len(res.Drinks) != 1, "expected 1 drink, got %d", len(res.Drinks))
	})
}
