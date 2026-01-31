package drinks_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestDrinks_ABAC_SommelierCannotChangeWineToCocktail(t *testing.T) {
	t.Parallel()

	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	base := b.WithIngredient("ABAC Base", measurement.UnitOz)

	owner := f.OwnerContext()
	sommelier := f.ActorContext("sommelier")

	wine := drinkForPolicy("House White", models.DrinkCategoryWine, base.ID)
	created, err := f.Drinks.Create(owner, &wine)
	testutil.Ok(t, err)

	updated := drinkForPolicy(created.Name, models.DrinkCategoryCocktail, base.ID)
	updated.ID = created.ID
	_, err = f.Drinks.Update(sommelier, &updated)
	testutil.PermissionTestFail(t, err)

	current, err := f.Drinks.Get(owner, created.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, current.Category != models.DrinkCategoryWine, "expected category to remain wine")
}

func TestDrinks_ABAC_BartenderCanUpdateCocktail(t *testing.T) {
	t.Parallel()

	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	base := b.WithIngredient("ABAC Base", measurement.UnitOz)

	owner := f.OwnerContext()
	bartender := f.ActorContext("bartender")

	cocktail := drinkForPolicy("Old Fashioned", models.DrinkCategoryCocktail, base.ID)
	created, err := f.Drinks.Create(owner, &cocktail)
	testutil.Ok(t, err)

	updated := drinkForPolicy(created.Name, models.DrinkCategoryCocktail, base.ID)
	updated.ID = created.ID
	updated.Description = "Stirred, not shaken"

	out, err := f.Drinks.Update(bartender, &updated)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, out.Category != models.DrinkCategoryCocktail, "expected cocktail category")
}
