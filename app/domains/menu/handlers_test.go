package menu_test

import (
	"testing"

	drinksM "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsM "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventoryM "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menuevents "github.com/TheFellow/go-modular-monolith/app/domains/menu/events"
	menudao "github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	menuM "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/mjl-/bstore"
)

func TestDrinkRecipeUpdatedMenuUpdater_MarksUnavailableWhenNewIngredientOutOfStock(t *testing.T) {
	f := testutil.NewFixture(t)

	base, err := f.Ingredients.Create(f.Ctx, ingredientsM.Ingredient{
		Name:     "Gin",
		Category: ingredientsM.CategorySpirit,
		Unit:     ingredientsM.UnitOz,
	})
	testutil.Ok(t, err)

	_, err = f.Inventory.Set(f.Ctx, inventoryM.Update{
		IngredientID: base.ID,
		Quantity:     10,
		CostPerUnit:  money.NewPriceFromCents(100, "USD"),
	})
	testutil.Ok(t, err)

	drink, err := f.Drinks.Create(f.Ctx, drinksM.Drink{
		Name:     "Gin Rickey",
		Category: drinksM.DrinkCategoryCocktail,
		Glass:    drinksM.GlassTypeRocks,
		Recipe: drinksM.Recipe{
			Ingredients: []drinksM.RecipeIngredient{
				{IngredientID: base.ID, Amount: 1, Unit: ingredientsM.UnitOz},
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

	rare, err := f.Ingredients.Create(f.Ctx, ingredientsM.Ingredient{
		Name:     "Rare Juice",
		Category: ingredientsM.CategoryJuice,
		Unit:     ingredientsM.UnitOz,
	})
	testutil.Ok(t, err)

	updated := *drink
	updated.Recipe.Ingredients = append(updated.Recipe.Ingredients, drinksM.RecipeIngredient{
		IngredientID: rare.ID,
		Amount:       1,
		Unit:         ingredientsM.UnitOz,
	})

	_, err = f.Drinks.Update(f.Ctx, updated)
	testutil.Ok(t, err)

	gotMenu, err := f.Menu.Get(f.Ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(gotMenu.Items) != 1, "expected 1 menu item, got %d", len(gotMenu.Items))
	testutil.ErrorIf(t, gotMenu.Items[0].Availability != menuM.AvailabilityUnavailable, "expected unavailable, got %s", gotMenu.Items[0].Availability)
}

func TestMenuPublishedValidator_SetsAvailabilityFromInventory(t *testing.T) {
	f := testutil.NewFixture(t)

	ingredient, err := f.Ingredients.Create(f.Ctx, ingredientsM.Ingredient{
		Name:     "Vodka",
		Category: ingredientsM.CategorySpirit,
		Unit:     ingredientsM.UnitOz,
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

	d := dispatcher.New()
	menuDAO := menudao.New()

	err = f.Store.Write(f.Ctx, func(tx *bstore.Tx) error {
		txCtx := middleware.NewContext(f.Ctx, middleware.WithTransaction(tx))

		updated := *menu
		updated.Status = menuM.MenuStatusPublished
		updated.Items[0].Availability = menuM.AvailabilityAvailable

		if err := menuDAO.Update(txCtx, updated); err != nil {
			return err
		}
		return d.Dispatch(txCtx, menuevents.MenuPublished{Menu: updated})
	})
	testutil.Ok(t, err)

	got, err := f.Menu.Get(f.Ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(got.Items) != 1, "expected 1 menu item, got %d", len(got.Items))
	testutil.ErrorIf(t, got.Items[0].Availability != menuM.AvailabilityUnavailable, "expected unavailable, got %s", got.Items[0].Availability)
}
