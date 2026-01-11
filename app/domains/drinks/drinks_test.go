package drinks_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsM "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestDrinks_CreateGetUpdateDelete(t *testing.T) {
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	lime := b.WithIngredient("Lime Juice", ingredientsM.UnitOz)
	lemon := b.WithIngredient("Lemon Juice", ingredientsM.UnitOz)

	created, err := f.Drinks.Create(ctx, models.Drink{
		Name:     "Margarita",
		Category: models.DrinkCategoryCocktail,
		Glass:    models.GlassTypeCoupe,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: lime.ID, Amount: 1.0, Unit: ingredientsM.UnitOz},
			},
			Steps: []string{"Shake with ice"},
		},
		Description: "A classic sour",
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, created.ID.ID == "", "expected id to be set")

	got, err := f.Drinks.Get(ctx, created.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, got.Name != "Margarita", "expected Margarita, got %q", got.Name)
	testutil.ErrorIf(t, len(got.Recipe.Ingredients) != 1, "expected 1 ingredient")
	testutil.ErrorIf(t, got.Recipe.Ingredients[0].IngredientID != lime.ID, "unexpected ingredient id")

	updated, err := f.Drinks.Update(ctx, models.Drink{
		ID:       created.ID,
		Name:     "Margarita",
		Category: models.DrinkCategoryCocktail,
		Glass:    models.GlassTypeCoupe,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: lemon.ID, Amount: 1.0, Unit: ingredientsM.UnitOz},
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
	testutil.ErrorIf(t, !errors.IsNotFound(err), "expected NotFound, got %v", err)
}

func TestDrinks_CreateRejectsIDProvided(t *testing.T) {
	f := testutil.NewFixture(t)

	_, err := f.Drinks.Create(f.OwnerContext(), models.Drink{
		ID: models.NewDrinkID("explicit-id"),
	})
	testutil.ErrorIf(t, err == nil || !errors.IsInvalid(err), "expected invalid error, got %v", err)
}

func TestDrinks_ListFiltersByName(t *testing.T) {
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	base := b.WithIngredient("Tequila", ingredientsM.UnitOz)

	b.WithDrink(models.Drink{
		Name:     "Margarita",
		Category: models.DrinkCategoryCocktail,
		Glass:    models.GlassTypeCoupe,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: base.ID, Amount: 2.0, Unit: ingredientsM.UnitOz},
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
				{IngredientID: base.ID, Amount: 1.5, Unit: ingredientsM.UnitOz},
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
				{IngredientID: base.ID, Amount: 2.0, Unit: ingredientsM.UnitOz},
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
