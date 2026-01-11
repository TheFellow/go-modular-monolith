package dispatcher_test

import (
	"testing"
	"time"

	drinksevents "github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	drinksM "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsM "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventoryevents "github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	inventoryM "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menuM "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/mjl-/bstore"
)

func TestDispatch_StockAdjusted_UpdatesMenuAvailability(t *testing.T) {
	f := testutil.NewFixture(t)
	a := f.App

	ingredient, err := a.Ingredients.Create(f.Ctx, ingredientsM.Ingredient{
		Name:     "Vodka",
		Category: ingredientsM.CategorySpirit,
		Unit:     ingredientsM.UnitOz,
	})
	if err != nil {
		t.Fatalf("create ingredient: %v", err)
	}

	_, err = a.Inventory.Set(f.Ctx, inventoryM.Update{
		IngredientID: ingredient.ID,
		Quantity:     10,
		CostPerUnit:  money.NewPriceFromCents(100, "USD"),
	})
	if err != nil {
		t.Fatalf("set stock: %v", err)
	}

	drink, err := a.Drinks.Create(f.Ctx, drinksM.Drink{
		Name:     "Margarita",
		Category: drinksM.DrinkCategoryCocktail,
		Glass:    drinksM.GlassTypeCoupe,
		Recipe: drinksM.Recipe{
			Ingredients: []drinksM.RecipeIngredient{
				{IngredientID: ingredient.ID, Amount: 1, Unit: ingredientsM.UnitOz},
			},
			Steps: []string{"Shake with ice"},
		},
	})
	if err != nil {
		t.Fatalf("create drink: %v", err)
	}

	m0, err := a.Menu.Create(f.Ctx, menuM.Menu{Name: "Happy Hour"})
	if err != nil {
		t.Fatalf("create menu: %v", err)
	}
	m1, err := a.Menu.AddDrink(f.Ctx, menuM.MenuDrinkChange{MenuID: m0.ID, DrinkID: drink.ID})
	if err != nil {
		t.Fatalf("add drink: %v", err)
	}
	m2, err := a.Menu.Publish(f.Ctx, menuM.Menu{ID: m1.ID})
	if err != nil {
		t.Fatalf("publish menu: %v", err)
	}
	if len(m2.Items) != 1 || m2.Items[0].Availability != menuM.AvailabilityAvailable {
		t.Fatalf("expected initial availability available, got %+v", m2.Items)
	}

	d := dispatcher.New()
	err = f.Store.Write(f.Ctx, func(tx *bstore.Tx) error {
		txCtx := middleware.NewContext(f.Ctx, middleware.WithTransaction(tx))
		return d.Dispatch(txCtx, inventoryevents.StockAdjusted{
			Previous: inventoryM.Inventory{IngredientID: ingredient.ID, Quantity: 10, Unit: ingredientsM.UnitOz},
			Current:  inventoryM.Inventory{IngredientID: ingredient.ID, Quantity: 0, Unit: ingredientsM.UnitOz},
			Reason:   "used",
		})
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	got, err := a.Menu.Get(f.Ctx, m2.ID)
	if err != nil {
		t.Fatalf("get menu: %v", err)
	}

	if len(got.Items) != 1 {
		t.Fatalf("expected 1 menu item, got %d", len(got.Items))
	}
	if got.Items[0].Availability != menuM.AvailabilityUnavailable {
		t.Fatalf("expected menu item availability unavailable, got %q", got.Items[0].Availability)
	}
}

func TestDispatch_DrinkDeleted_RemovesMenuItems(t *testing.T) {
	f := testutil.NewFixture(t)
	a := f.App

	// Create an ingredient to make recipes valid
	ingredient, err := a.Ingredients.Create(f.Ctx, ingredientsM.Ingredient{
		Name:     "Vodka",
		Category: ingredientsM.CategorySpirit,
		Unit:     ingredientsM.UnitOz,
	})
	if err != nil {
		t.Fatalf("create ingredient: %v", err)
	}

	// Create two drinks
	drink1, err := a.Drinks.Create(f.Ctx, drinksM.Drink{
		Name:     "Martini",
		Category: drinksM.DrinkCategoryCocktail,
		Glass:    drinksM.GlassTypeMartini,
		Recipe: drinksM.Recipe{
			Ingredients: []drinksM.RecipeIngredient{
				{IngredientID: ingredient.ID, Amount: 2, Unit: ingredientsM.UnitOz},
			},
			Steps: []string{"Stir with ice"},
		},
	})
	if err != nil {
		t.Fatalf("create drink1: %v", err)
	}

	drink2, err := a.Drinks.Create(f.Ctx, drinksM.Drink{
		Name:     "Cosmopolitan",
		Category: drinksM.DrinkCategoryCocktail,
		Glass:    drinksM.GlassTypeMartini,
		Recipe: drinksM.Recipe{
			Ingredients: []drinksM.RecipeIngredient{
				{IngredientID: ingredient.ID, Amount: 1.5, Unit: ingredientsM.UnitOz},
			},
			Steps: []string{"Shake with ice"},
		},
	})
	if err != nil {
		t.Fatalf("create drink2: %v", err)
	}

	// Create a menu with both drinks
	m0, err := a.Menu.Create(f.Ctx, menuM.Menu{Name: "Cocktail Menu"})
	if err != nil {
		t.Fatalf("create menu: %v", err)
	}

	m1, err := a.Menu.AddDrink(f.Ctx, menuM.MenuDrinkChange{MenuID: m0.ID, DrinkID: drink1.ID})
	if err != nil {
		t.Fatalf("add drink1: %v", err)
	}

	m2, err := a.Menu.AddDrink(f.Ctx, menuM.MenuDrinkChange{MenuID: m1.ID, DrinkID: drink2.ID})
	if err != nil {
		t.Fatalf("add drink2: %v", err)
	}

	if len(m2.Items) != 2 {
		t.Fatalf("expected 2 menu items, got %d", len(m2.Items))
	}

	// Dispatch DrinkDeleted event for drink1
	d := dispatcher.New()
	err = f.Store.Write(f.Ctx, func(tx *bstore.Tx) error {
		txCtx := middleware.NewContext(f.Ctx, middleware.WithTransaction(tx))
		return d.Dispatch(txCtx, drinksevents.DrinkDeleted{Drink: *drink1, DeletedAt: time.Now().UTC()})
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	// Verify menu now has only drink2
	got, err := a.Menu.Get(f.Ctx, m2.ID)
	if err != nil {
		t.Fatalf("get menu: %v", err)
	}

	if len(got.Items) != 1 {
		t.Fatalf("expected 1 menu item after delete, got %d", len(got.Items))
	}

	if string(got.Items[0].DrinkID.ID) != string(drink2.ID.ID) {
		t.Fatalf("expected remaining item to be drink2, got %s", string(got.Items[0].DrinkID.ID))
	}
}
