package ingredients_test

import (
	"testing"

	drinksM "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsM "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventoryM "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menuM "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestIngredients_Delete_CascadesToDrinksMenusAndInventory(t *testing.T) {
	f := testutil.NewFixture(t)

	ingredient, err := f.Ingredients.Create(f.Ctx, ingredientsM.Ingredient{
		Name:     "Vodka",
		Category: ingredientsM.CategorySpirit,
		Unit:     ingredientsM.UnitOz,
	})
	testutil.Ok(t, err)

	_, err = f.Inventory.Set(f.Ctx, inventoryM.Update{
		IngredientID: ingredient.ID,
		Quantity:     10,
		CostPerUnit:  money.NewPriceFromCents(100, "USD"),
	})
	testutil.Ok(t, err)

	drink, err := f.Drinks.Create(f.Ctx, drinksM.Drink{
		Name:     "Vodka Soda",
		Category: drinksM.DrinkCategoryCocktail,
		Glass:    drinksM.GlassTypeRocks,
		Recipe: drinksM.Recipe{
			Ingredients: []drinksM.RecipeIngredient{
				{IngredientID: ingredient.ID, Amount: 1, Unit: ingredientsM.UnitOz},
			},
			Steps: []string{"build"},
		},
	})
	testutil.Ok(t, err)

	menu, err := f.Menu.Create(f.Ctx, menuM.Menu{Name: "Test Menu"})
	testutil.Ok(t, err)
	menu, err = f.Menu.AddDrink(f.Ctx, menuM.MenuDrinkChange{MenuID: menu.ID, DrinkID: drink.ID})
	testutil.Ok(t, err)
	menu, err = f.Menu.Publish(f.Ctx, menuM.Menu{ID: menu.ID})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(menu.Items) != 1, "expected 1 menu item, got %d", len(menu.Items))

	_, err = f.Ingredients.Delete(f.Ctx, ingredient.ID)
	testutil.Ok(t, err)

	_, err = f.Inventory.Get(f.Ctx, ingredient.ID)
	testutil.ErrorIf(t, !errors.IsNotFound(err), "expected stock not found, got %v", err)

	_, err = f.Drinks.Get(f.Ctx, drink.ID)
	testutil.ErrorIf(t, !errors.IsNotFound(err), "expected drink not found, got %v", err)

	gotMenu, err := f.Menu.Get(f.Ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(gotMenu.Items) != 0, "expected menu items to be removed, got %d", len(gotMenu.Items))
}
