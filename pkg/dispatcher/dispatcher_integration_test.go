package dispatcher_test

import (
	"testing"
	"time"

	drinksevents "github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	drinksM "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsM "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventoryM "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menuM "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/mjl-/bstore"
)

func TestDispatch_StockAdjusted_UpdatesMenuAvailability(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	a := f.App
	ctx := f.OwnerContext()

	ingredient, err := a.Ingredients.Create(ctx, &ingredientsM.Ingredient{
		Name:     "Vodka",
		Category: ingredientsM.CategorySpirit,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)

	_, err = a.Inventory.Set(ctx, &inventoryM.Update{
		IngredientID: ingredient.ID,
		Amount:       measurement.MustAmount(10, ingredient.Unit),
		CostPerUnit:  money.NewPriceFromCents(100, currency.USD),
	})
	testutil.Ok(t, err)

	drink, err := a.Drinks.Create(ctx, &drinksM.Drink{
		Name:     "Margarita",
		Category: drinksM.DrinkCategoryCocktail,
		Glass:    drinksM.GlassTypeCoupe,
		Recipe: drinksM.Recipe{
			Ingredients: []drinksM.RecipeIngredient{
				{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)},
			},
			Steps: []string{"Shake with ice"},
		},
	})
	testutil.Ok(t, err)

	m0, err := a.Menus.Create(ctx, &menuM.Menu{Name: "Happy Hour"})
	testutil.Ok(t, err)
	m1, err := a.Menus.AddDrink(ctx, &menuM.MenuPatch{MenuID: m0.ID, DrinkID: drink.ID})
	testutil.Ok(t, err)
	m2, err := a.Menus.Publish(ctx, &menuM.Menu{ID: m1.ID})
	testutil.Ok(t, err)
	testutil.Equals(t, len(m2.Items), 1)
	testutil.Equals(t, m2.Items[0].Availability, menuM.AvailabilityAvailable)

	_, err = a.Inventory.Set(ctx, &inventoryM.Update{
		IngredientID: ingredient.ID,
		Amount:       measurement.MustAmount(0, ingredient.Unit),
		CostPerUnit:  money.NewPriceFromCents(100, currency.USD),
	})
	testutil.Ok(t, err)

	got, err := a.Menus.Get(ctx, m2.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(got.Items), 1)
	testutil.Equals(t, got.Items[0].Availability, menuM.AvailabilityUnavailable)
}

func TestDispatch_DrinkDeleted_RemovesMenuItems(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	a := f.App
	ctx := f.OwnerContext()

	// Create an ingredient to make recipes valid
	ingredient, err := a.Ingredients.Create(ctx, &ingredientsM.Ingredient{
		Name:     "Vodka",
		Category: ingredientsM.CategorySpirit,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)

	// Create two drinks
	drink1, err := a.Drinks.Create(ctx, &drinksM.Drink{
		Name:     "Martini",
		Category: drinksM.DrinkCategoryCocktail,
		Glass:    drinksM.GlassTypeMartini,
		Recipe: drinksM.Recipe{
			Ingredients: []drinksM.RecipeIngredient{
				{IngredientID: ingredient.ID, Amount: measurement.MustAmount(2, measurement.UnitOz)},
			},
			Steps: []string{"Stir with ice"},
		},
	})
	testutil.Ok(t, err)

	drink2, err := a.Drinks.Create(ctx, &drinksM.Drink{
		Name:     "Cosmopolitan",
		Category: drinksM.DrinkCategoryCocktail,
		Glass:    drinksM.GlassTypeMartini,
		Recipe: drinksM.Recipe{
			Ingredients: []drinksM.RecipeIngredient{
				{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1.5, measurement.UnitOz)},
			},
			Steps: []string{"Shake with ice"},
		},
	})
	testutil.Ok(t, err)

	// Create a menu with both drinks
	m0, err := a.Menus.Create(ctx, &menuM.Menu{Name: "Cocktail Menu"})
	testutil.Ok(t, err)

	m1, err := a.Menus.AddDrink(ctx, &menuM.MenuPatch{MenuID: m0.ID, DrinkID: drink1.ID})
	testutil.Ok(t, err)

	m2, err := a.Menus.AddDrink(ctx, &menuM.MenuPatch{MenuID: m1.ID, DrinkID: drink2.ID})
	testutil.Ok(t, err)
	testutil.Equals(t, len(m2.Items), 2)

	// Dispatch DrinkDeleted event for drink1
	d := dispatcher.New(f.Store)
	err = f.Store.Write(ctx, func(tx *bstore.Tx) error {
		txCtx := ctx.WithTransaction(tx)
		return d.Dispatch(txCtx, drinksevents.DrinkDeleted{Drink: *drink1, DeletedAt: time.Now().UTC()})
	})
	testutil.Ok(t, err)

	// Verify menu now has only drink2
	got, err := a.Menus.Get(ctx, m2.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(got.Items), 1)
	testutil.Equals(t, got.Items[0].DrinkID, drink2.ID)
}
