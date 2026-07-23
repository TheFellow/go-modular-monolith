package drinks_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestDrinks_CreateGetUpdateDelete(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	lime := b.WithIngredient("Lime Juice", measurement.UnitOz)
	lemon := b.WithIngredient("Lemon Juice", measurement.UnitOz)

	created, err := f.Drinks.Create(ctx, &models.Drink{
		Name:     "Margarita",
		Category: models.DrinkCategoryCocktail,
		Glass:    models.GlassTypeCoupe,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: lime.ID, Amount: measurement.MustAmount(1.0, measurement.UnitOz)},
			},
			Steps: []string{"Shake with ice"},
		},
		Description: "A classic sour",
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, created.ID.IsZero(), "expected id to be set")

	got, err := f.Drinks.Get(ctx, created.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, got.Name != "Margarita", "expected Margarita, got %q", got.Name)
	testutil.ErrorIf(t, len(got.Recipe.Ingredients) != 1, "expected 1 ingredient")
	testutil.ErrorIf(t, got.Recipe.Ingredients[0].IngredientID != lime.ID, "unexpected ingredient id")

	updated, err := f.Drinks.Update(ctx, &models.Drink{
		ID:       created.ID,
		Name:     "Margarita",
		Category: models.DrinkCategoryCocktail,
		Glass:    models.GlassTypeCoupe,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: lemon.ID, Amount: measurement.MustAmount(1.0, measurement.UnitOz)},
			},
			Steps: []string{"Shake hard"},
		},
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, updated.Recipe.Ingredients[0].IngredientID != lemon.ID, "expected lemon ingredient")

	got, err = f.Drinks.Get(ctx, created.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, got.Recipe.Ingredients[0].IngredientID != lemon.ID, "expected lemon ingredient after update")

	deleted, err := f.Drinks.Delete(ctx, created.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, !deleted.DeletedAt.IsSome(), "expected DeletedAt to be set")

	_, err = f.Drinks.Get(ctx, created.ID)
	testutil.ErrorIsNotFound(t, err)
}
