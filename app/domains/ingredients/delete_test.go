package ingredients_test

import (
	"math"
	"testing"

	drinksM "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsM "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventoryM "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menuM "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestIngredients_Delete_CascadesToDrinksMenusAndInventory(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	ingredient, err := f.Ingredients.Create(ctx, &ingredientsM.Ingredient{
		Name:     "Vodka",
		Category: ingredientsM.CategorySpirit,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)

	_, err = f.Inventory.Set(ctx, &inventoryM.Update{
		IngredientID: ingredient.ID,
		Amount:       measurement.MustAmount(10, ingredient.Unit),
		CostPerUnit:  money.NewPriceFromCents(100, currency.USD),
	})
	testutil.Ok(t, err)

	drink, err := f.Drinks.Create(ctx, &drinksM.Drink{
		Name:     "Vodka Soda",
		Category: drinksM.DrinkCategoryCocktail,
		Glass:    drinksM.GlassTypeRocks,
		Recipe: drinksM.Recipe{
			Ingredients: []drinksM.RecipeIngredient{
				{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)},
			},
			Steps: []string{"build"},
		},
	})
	testutil.Ok(t, err)

	menu, err := f.Menus.Create(ctx, &menuM.Menu{Name: "Test Menu"})
	testutil.Ok(t, err)
	menu, err = f.Menus.AddDrink(ctx, &menuM.MenuPatch{MenuID: menu.ID, DrinkID: drink.ID})
	testutil.Ok(t, err)
	menu, err = f.Menus.Publish(ctx, &menuM.Menu{ID: menu.ID})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(menu.Items) != 1, "expected 1 menu item, got %d", len(menu.Items))

	_, err = f.Ingredients.Delete(ctx, ingredient.ID)
	testutil.Ok(t, err)

	_, err = f.Inventory.Get(ctx, ingredient.ID)
	testutil.ErrorIsNotFound(t, err)

	gotDrink, err := f.Drinks.Get(ctx, drink.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotDrink.Status, drinksM.StatusReviewRequired)

	gotMenu, err := f.Menus.Get(ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(gotMenu.Items), 1)
	testutil.Equals(t, gotMenu.Items[0].Availability, menuM.AvailabilityUnavailable)
}

func TestIngredients_RetireRejectsIncompatibleReplacement(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()
	spirit := testutil.CreateIngredient(t, f, ingredientsM.Ingredient{Name: "Spirit", Category: ingredientsM.CategorySpirit, Unit: measurement.UnitOz})
	garnish := testutil.CreateIngredient(t, f, ingredientsM.Ingredient{Name: "Garnish", Category: ingredientsM.CategoryGarnish, Unit: measurement.UnitPiece})

	_, err := f.Ingredients.Retire(ctx, spirit.ID, ingredientsM.Retirement{ReplacementID: garnish.ID, Ratio: 1})
	testutil.ErrorIf(t, err == nil, "expected incompatible replacement error")
	_, getErr := f.Ingredients.Get(ctx, spirit.ID)
	testutil.Ok(t, getErr)

	compatible := testutil.CreateIngredient(t, f, ingredientsM.Ingredient{Name: "Compatible", Category: ingredientsM.CategorySpirit, Unit: measurement.UnitOz})
	for _, ratio := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, ratioErr := f.Ingredients.Retire(ctx, spirit.ID, ingredientsM.Retirement{ReplacementID: compatible.ID, Ratio: ratio})
		testutil.ErrorIf(t, ratioErr == nil, "non-finite replacement ratio accepted")
	}
}
