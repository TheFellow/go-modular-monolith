package ingredients_test

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestIngredients_Update_AllowsUnitChange(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	ingredient, err := f.Ingredients.Create(ctx, &models.Ingredient{
		Name:     "Simple Syrup",
		Category: models.CategorySyrup,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)

	updated, err := f.Ingredients.Update(ctx, &models.Ingredient{
		ID:   ingredient.ID,
		Unit: measurement.UnitMl,
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, updated.Unit != measurement.UnitMl, "expected unit ml, got %s", updated.Unit)
}

func TestIngredients_Update_AllowsUnitChangeWhenUsed(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	ingredient, err := f.Ingredients.Create(ctx, &models.Ingredient{
		Name:     "Gin",
		Category: models.CategorySpirit,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)

	_, err = f.Drinks.Create(ctx, &drinksmodels.Drink{
		Name:     "Gin & Tonic",
		Category: drinksmodels.DrinkCategoryHighball,
		Glass:    drinksmodels.GlassTypeHighball,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{IngredientID: ingredient.ID, Amount: measurement.MustAmount(2, measurement.UnitOz)},
			},
			Steps: []string{"Build"},
		},
	})
	testutil.Ok(t, err)

	updated, err := f.Ingredients.Update(ctx, &models.Ingredient{
		ID:   ingredient.ID,
		Unit: measurement.UnitMl,
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, updated.Unit != measurement.UnitMl, "expected unit ml, got %s", updated.Unit)
}

func TestIngredients_Update_AllowsOtherFieldsWhenUsed(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	ingredient, err := f.Ingredients.Create(ctx, &models.Ingredient{
		Name:     "Lime Juice",
		Category: models.CategoryJuice,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)

	_, err = f.Drinks.Create(ctx, &drinksmodels.Drink{
		Name:     "Gimlet",
		Category: drinksmodels.DrinkCategoryCocktail,
		Glass:    drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)},
			},
			Steps: []string{"Shake"},
		},
	})
	testutil.Ok(t, err)

	updated, err := f.Ingredients.Update(ctx, &models.Ingredient{
		ID:   ingredient.ID,
		Name: "Fresh Lime Juice",
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, updated.Name != "Fresh Lime Juice", "expected updated name, got %q", updated.Name)
	testutil.ErrorIf(t, updated.Unit != ingredient.Unit, "expected unit to remain %s, got %s", ingredient.Unit, updated.Unit)
}
