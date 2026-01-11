package menu_test

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menuevents "github.com/TheFellow/go-modular-monolith/app/domains/menu/events"
	menudao "github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/mjl-/bstore"
)

func TestDrinkRecipeUpdatedMenuUpdater_MarksUnavailableWhenNewIngredientOutOfStock(t *testing.T) {
	fix := testutil.NewFixture(t)
	ctx := fix.Ctx

	base, err := fix.Ingredients.Create(ctx, ingredientsmodels.Ingredient{
		Name:     "Gin",
		Category: ingredientsmodels.CategorySpirit,
		Unit:     ingredientsmodels.UnitOz,
	})
	testutil.Ok(t, err)

	_, err = fix.Inventory.Set(ctx, inventorymodels.Update{
		IngredientID: base.ID,
		Quantity:     10,
		CostPerUnit:  money.NewPriceFromCents(100, "USD"),
	})
	testutil.Ok(t, err)

	drink, err := fix.Drinks.Create(ctx, drinksmodels.Drink{
		Name:     "Gin Rickey",
		Category: drinksmodels.DrinkCategoryCocktail,
		Glass:    drinksmodels.GlassTypeRocks,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{IngredientID: base.ID, Amount: 1, Unit: ingredientsmodels.UnitOz},
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

	rare, err := fix.Ingredients.Create(ctx, ingredientsmodels.Ingredient{
		Name:     "Rare Juice",
		Category: ingredientsmodels.CategoryJuice,
		Unit:     ingredientsmodels.UnitOz,
	})
	testutil.Ok(t, err)

	updated := *drink
	updated.Recipe.Ingredients = append(updated.Recipe.Ingredients, drinksmodels.RecipeIngredient{
		IngredientID: rare.ID,
		Amount:       1,
		Unit:         ingredientsmodels.UnitOz,
	})

	_, err = fix.Drinks.Update(ctx, updated)
	testutil.Ok(t, err)

	gotMenu, err := fix.Menu.Get(ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(gotMenu.Items) != 1, "expected 1 menu item, got %d", len(gotMenu.Items))
	testutil.ErrorIf(t, gotMenu.Items[0].Availability != menumodels.AvailabilityUnavailable, "expected unavailable, got %s", gotMenu.Items[0].Availability)
}

func TestMenuPublishedValidator_SetsAvailabilityFromInventory(t *testing.T) {
	fix := testutil.NewFixture(t)
	ctx := fix.Ctx

	ingredient, err := fix.Ingredients.Create(ctx, ingredientsmodels.Ingredient{
		Name:     "Vodka",
		Category: ingredientsmodels.CategorySpirit,
		Unit:     ingredientsmodels.UnitOz,
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

	d := dispatcher.New()
	menuDAO := menudao.New()

	err = fix.Store.Write(ctx, func(tx *bstore.Tx) error {
		txCtx := middleware.NewContext(ctx, middleware.WithTransaction(tx))

		updated := *menu
		updated.Status = menumodels.MenuStatusPublished
		updated.Items[0].Availability = menumodels.AvailabilityAvailable

		if err := menuDAO.Update(txCtx, updated); err != nil {
			return err
		}
		return d.Dispatch(txCtx, menuevents.MenuPublished{Menu: updated})
	})
	testutil.Ok(t, err)

	got, err := fix.Menu.Get(ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(got.Items) != 1, "expected 1 menu item, got %d", len(got.Items))
	testutil.ErrorIf(t, got.Items[0].Availability != menumodels.AvailabilityUnavailable, "expected unavailable, got %s", got.Items[0].Availability)
}
