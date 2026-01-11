package ingredients_test

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestIngredients_Delete_CascadesToDrinksMenusAndInventory(t *testing.T) {
	fix := testutil.NewFixture(t)
	ctx := fix.Ctx

	ingredient, err := fix.Ingredients.Create(ctx, ingredientsmodels.Ingredient{
		Name:     "Vodka",
		Category: ingredientsmodels.CategorySpirit,
		Unit:     ingredientsmodels.UnitOz,
	})
	testutil.Ok(t, err)

	_, err = fix.Inventory.Set(ctx, inventorymodels.StockUpdate{
		IngredientID: ingredient.ID,
		Quantity:     10,
		CostPerUnit:  money.NewPriceFromCents(100, "USD"),
	})
	testutil.Ok(t, err)

	drink, err := fix.Drinks.Create(ctx, drinksmodels.Drink{
		Name:     "Vodka Soda",
		Category: drinksmodels.DrinkCategoryCocktail,
		Glass:    drinksmodels.GlassTypeRocks,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{IngredientID: ingredient.ID, Amount: 1, Unit: ingredientsmodels.UnitOz},
			},
			Steps: []string{"build"},
		},
	})
	testutil.Ok(t, err)

	menu, err := fix.Menu.Create(ctx, menumodels.Menu{Name: "Test Menu"})
	testutil.Ok(t, err)
	menu, err = fix.Menu.AddDrink(ctx, menumodels.MenuDrinkChange{MenuID: menu.ID, DrinkID: drink.ID})
	testutil.Ok(t, err)
	menu, err = fix.Menu.Publish(ctx, menumodels.Menu{ID: menu.ID})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(menu.Items) != 1, "expected 1 menu item, got %d", len(menu.Items))

	_, err = fix.Ingredients.Delete(ctx, ingredient.ID)
	testutil.Ok(t, err)

	_, err = fix.Inventory.Get(ctx, ingredient.ID)
	testutil.ErrorIf(t, !errors.IsNotFound(err), "expected stock not found, got %v", err)

	_, err = fix.Drinks.Get(ctx, drink.ID)
	testutil.ErrorIf(t, !errors.IsNotFound(err), "expected drink not found, got %v", err)

	gotMenu, err := fix.Menu.Get(ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(gotMenu.Items) != 0, "expected menu items to be removed, got %d", len(gotMenu.Items))
}
