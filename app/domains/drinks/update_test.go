package drinks_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestDrinks_ABAC_SommelierCannotChangeWineToCocktail(t *testing.T) {
	t.Parallel()

	f := testutil.NewFixture(t)
	base := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "ABAC Base", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
	})

	owner := f.OwnerContext()
	sommelier := f.ActorContext("sommelier")

	wine := drinkForPolicy("House White", models.DrinkCategoryWine, base.ID)
	created, err := f.Drinks.Create(owner, &wine)
	testutil.Ok(t, err)

	updated := drinkForPolicy(created.Name, models.DrinkCategoryCocktail, base.ID)
	updated.ID = created.ID
	updated.Revision = created.Revision
	_, err = f.Drinks.Update(sommelier, &updated)
	testutil.ErrorIsPermission(t, err)

	current, err := f.Drinks.Get(owner, created.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, current.Category != models.DrinkCategoryWine, "expected category to remain wine")
}

func TestDrinks_ABAC_BartenderCanUpdateCocktail(t *testing.T) {
	t.Parallel()

	f := testutil.NewFixture(t)
	base := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "ABAC Base", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
	})

	owner := f.OwnerContext()
	bartender := f.ActorContext("bartender")

	cocktail := drinkForPolicy("Old Fashioned", models.DrinkCategoryCocktail, base.ID)
	created, err := f.Drinks.Create(owner, &cocktail)
	testutil.Ok(t, err)

	updated := drinkForPolicy(created.Name, models.DrinkCategoryCocktail, base.ID)
	updated.ID = created.ID
	updated.Revision = created.Revision
	updated.Description = "Stirred, not shaken"

	out, err := f.Drinks.Update(bartender, &updated)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, out.Category != models.DrinkCategoryCocktail, "expected cocktail category")
}

func TestDrinks_UpdateRejectsStaleRevision(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()
	base := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Revision Base", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
	})

	created, err := f.Drinks.Create(ctx, new(drinkForPolicy("Original", models.DrinkCategoryCocktail, base.ID)))
	testutil.Ok(t, err)
	winner, stale := *created, *created
	winner.Description = "winner"
	committed, err := f.Drinks.Update(ctx, &winner)
	testutil.Ok(t, err)
	testutil.Equals(t, committed.Revision, created.Revision+1)

	stale.Description = "stale"
	_, err = f.Drinks.Update(ctx, &stale)
	testutil.ErrorIsConflict(t, err)
	current, err := f.Drinks.Get(ctx, created.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, current.Description, "winner")
}
