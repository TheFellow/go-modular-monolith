package drinks_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestDrinks_ListFiltersByName(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	base := b.WithIngredient("Tequila", measurement.UnitOz)

	b.WithDrink(models.Drink{
		Name:     "Margarita",
		Category: models.DrinkCategoryCocktail,
		Glass:    models.GlassTypeCoupe,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: base.ID, Amount: measurement.MustAmount(2.0, measurement.UnitOz)},
			},
			Steps: []string{"Shake"},
		},
	})
	b.WithDrink(models.Drink{
		Name:     "Cosmopolitan",
		Category: models.DrinkCategoryCocktail,
		Glass:    models.GlassTypeMartini,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: base.ID, Amount: measurement.MustAmount(1.5, measurement.UnitOz)},
			},
			Steps: []string{"Shake"},
		},
	})
	b.WithDrink(models.Drink{
		Name:     "Old Fashioned",
		Category: models.DrinkCategoryCocktail,
		Glass:    models.GlassTypeRocks,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: base.ID, Amount: measurement.MustAmount(2.0, measurement.UnitOz)},
			},
			Steps: []string{"Stir"},
		},
	})

	all, err := f.Drinks.List(ctx, drinks.ListRequest{})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(all) != 3, "expected 3 drinks, got %d", len(all))

	filtered, err := f.Drinks.List(ctx, drinks.ListRequest{Name: "Margarita"})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(filtered) != 1, "expected 1 drink, got %d", len(filtered))
	testutil.ErrorIf(t, filtered[0].Name != "Margarita", "expected Margarita, got %q", filtered[0].Name)
}
