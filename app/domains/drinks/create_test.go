package drinks_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestDrinks_CreateRejectsIDProvided(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	_, err := f.Drinks.Create(f.OwnerContext(), &models.Drink{
		ID: models.NewDrinkID("explicit-id"),
	})
	testutil.ErrorIsInvalid(t, err)
}

func TestDrinks_ABAC_SommelierCanCreateWine(t *testing.T) {
	t.Parallel()

	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	base := b.WithIngredient("ABAC Base", measurement.UnitOz)

	sommelier := f.ActorContext("sommelier")

	wine := drinkForPolicy("House Red", models.DrinkCategoryWine, base.ID)
	created, err := f.Drinks.Create(sommelier, &wine)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, created.Category != models.DrinkCategoryWine, "expected wine category")

	cocktail := drinkForPolicy("Negroni", models.DrinkCategoryCocktail, base.ID)
	_, err = f.Drinks.Create(sommelier, &cocktail)
	testutil.PermissionTestFail(t, err)
}
