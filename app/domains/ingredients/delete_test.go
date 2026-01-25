package ingredients_test

import (
	"testing"

	drinksM "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsM "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventoryM "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menuM "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
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

	menu, err := f.Menu.Create(ctx, &menuM.Menu{Name: "Test Menu"})
	testutil.Ok(t, err)
	menu, err = f.Menu.AddDrink(ctx, &menuM.MenuDrinkChange{MenuID: menu.ID, DrinkID: drink.ID})
	testutil.Ok(t, err)
	menu, err = f.Menu.Publish(ctx, &menuM.Menu{ID: menu.ID})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(menu.Items) != 1, "expected 1 menu item, got %d", len(menu.Items))

	_, err = f.Ingredients.Delete(ctx, ingredient.ID)
	testutil.Ok(t, err)

	_, err = f.Inventory.Get(ctx, ingredient.ID)
	testutil.ErrorIsNotFound(t, err)

	_, err = f.Drinks.Get(ctx, drink.ID)
	testutil.ErrorIsNotFound(t, err)

	gotMenu, err := f.Menu.Get(ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(gotMenu.Items) != 0, "expected menu items to be removed, got %d", len(gotMenu.Items))
}
