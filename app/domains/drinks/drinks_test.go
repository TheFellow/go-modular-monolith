package drinks_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestDrinks_CreateGetUpdateDelete(t *testing.T) {
	fix := testutil.NewFixture(t)
	b := fix.Bootstrap()

	lime := b.WithIngredient("Lime Juice", ingredientsmodels.UnitOz)
	lemon := b.WithIngredient("Lemon Juice", ingredientsmodels.UnitOz)

	created, err := fix.Drinks.Create(fix.Ctx, models.Drink{
		Name:     "Margarita",
		Category: models.DrinkCategoryCocktail,
		Glass:    models.GlassTypeCoupe,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: lime.ID, Amount: 1.0, Unit: ingredientsmodels.UnitOz},
			},
			Steps: []string{"Shake with ice"},
		},
		Description: "A classic sour",
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, created.ID.ID == "", "expected id to be set")

	got, err := fix.Drinks.Get(fix.Ctx, drinks.GetRequest{ID: created.ID})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, got.Drink.Name != "Margarita", "expected Margarita, got %q", got.Drink.Name)
	testutil.ErrorIf(t, len(got.Drink.Recipe.Ingredients) != 1, "expected 1 ingredient")
	testutil.ErrorIf(t, got.Drink.Recipe.Ingredients[0].IngredientID != lime.ID, "unexpected ingredient id")

	updated, err := fix.Drinks.Update(fix.Ctx, models.Drink{
		ID:       created.ID,
		Name:     "Margarita",
		Category: models.DrinkCategoryCocktail,
		Glass:    models.GlassTypeCoupe,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: lemon.ID, Amount: 1.0, Unit: ingredientsmodels.UnitOz},
			},
			Steps: []string{"Shake hard"},
		},
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, updated.Recipe.Ingredients[0].IngredientID != lemon.ID, "expected lemon ingredient")

	got, err = fix.Drinks.Get(fix.Ctx, drinks.GetRequest{ID: created.ID})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, got.Drink.Recipe.Ingredients[0].IngredientID != lemon.ID, "expected lemon ingredient after update")

	_, err = fix.Drinks.Delete(fix.Ctx, drinks.DeleteRequest{ID: created.ID})
	testutil.Ok(t, err)

	_, err = fix.Drinks.Get(fix.Ctx, drinks.GetRequest{ID: created.ID})
	testutil.ErrorIf(t, !errors.IsNotFound(err), "expected NotFound, got %v", err)
}

func TestDrinks_CreateRejectsIDProvided(t *testing.T) {
	fix := testutil.NewFixture(t)

	_, err := fix.Drinks.Create(fix.Ctx, models.Drink{
		ID: models.NewDrinkID("explicit-id"),
	})
	testutil.ErrorIf(t, err == nil || !errors.IsInvalid(err), "expected invalid error, got %v", err)
}

func TestDrinks_ListFiltersByName(t *testing.T) {
	fix := testutil.NewFixture(t)
	b := fix.Bootstrap()

	base := b.WithIngredient("Tequila", ingredientsmodels.UnitOz)

	b.WithDrink(models.Drink{
		Name:     "Margarita",
		Category: models.DrinkCategoryCocktail,
		Glass:    models.GlassTypeCoupe,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: base.ID, Amount: 2.0, Unit: ingredientsmodels.UnitOz},
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
				{IngredientID: base.ID, Amount: 1.5, Unit: ingredientsmodels.UnitOz},
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
				{IngredientID: base.ID, Amount: 2.0, Unit: ingredientsmodels.UnitOz},
			},
			Steps: []string{"Stir"},
		},
	})

	all, err := fix.Drinks.List(fix.Ctx, drinks.ListRequest{})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(all.Drinks) != 3, "expected 3 drinks, got %d", len(all.Drinks))

	filtered, err := fix.Drinks.List(fix.Ctx, drinks.ListRequest{Name: "Margarita"})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(filtered.Drinks) != 1, "expected 1 drink, got %d", len(filtered.Drinks))
	testutil.ErrorIf(t, filtered.Drinks[0].Name != "Margarita", "expected Margarita, got %q", filtered.Drinks[0].Name)
}
