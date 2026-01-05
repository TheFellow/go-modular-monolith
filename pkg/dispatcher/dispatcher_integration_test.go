package dispatcher_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/money"
	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/mjl-/bstore"
)

func TestDispatch_StockAdjusted_UpdatesMenuAvailability(t *testing.T) {
	fix := testutil.NewFixture(t)
	a := fix.App
	ctx := fix.Ctx

	ingredient, err := a.Ingredients.Create(ctx, ingredientsmodels.Ingredient{
		Name:     "Vodka",
		Category: ingredientsmodels.CategorySpirit,
		Unit:     ingredientsmodels.UnitOz,
	})
	if err != nil {
		t.Fatalf("create ingredient: %v", err)
	}

	_, err = a.Inventory.Set(ctx, inventorymodels.StockUpdate{
		IngredientID: ingredient.ID,
		Quantity:     10,
		CostPerUnit:  money.Price{Amount: 100, Currency: "USD"},
	})
	if err != nil {
		t.Fatalf("set stock: %v", err)
	}

	drink, err := a.Drinks.Create(ctx, models.Drink{
		Name:     "Margarita",
		Category: models.DrinkCategoryCocktail,
		Glass:    models.GlassTypeCoupe,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: ingredient.ID, Amount: 1, Unit: ingredientsmodels.UnitOz},
			},
			Steps: []string{"Shake with ice"},
		},
	})
	if err != nil {
		t.Fatalf("create drink: %v", err)
	}

	m0, err := a.Menu.Create(ctx, menumodels.Menu{Name: "Happy Hour"})
	if err != nil {
		t.Fatalf("create menu: %v", err)
	}
	m1, err := a.Menu.AddDrink(ctx, menumodels.MenuDrinkChange{MenuID: m0.ID, DrinkID: drink.ID})
	if err != nil {
		t.Fatalf("add drink: %v", err)
	}
	m2, err := a.Menu.Publish(ctx, menumodels.Menu{ID: m1.ID})
	if err != nil {
		t.Fatalf("publish menu: %v", err)
	}
	if len(m2.Items) != 1 || m2.Items[0].Availability != menumodels.AvailabilityAvailable {
		t.Fatalf("expected initial availability available, got %+v", m2.Items)
	}

	d := dispatcher.New()
	err = fix.Store.Write(ctx, func(tx *bstore.Tx) error {
		txCtx := middleware.NewContext(ctx, middleware.WithTransaction(tx))
		return d.Dispatch(txCtx, events.StockAdjusted{
			IngredientID: ingredient.ID,
			PreviousQty:  10,
			NewQty:       0,
			Delta:        -10,
			Reason:       "used",
		})
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	got, err := a.Menu.Get(ctx, menu.GetRequest{ID: m2.ID})
	if err != nil {
		t.Fatalf("get menu: %v", err)
	}

	if len(got.Menu.Items) != 1 {
		t.Fatalf("expected 1 menu item, got %d", len(got.Menu.Items))
	}
	if got.Menu.Items[0].Availability != menumodels.AvailabilityUnavailable {
		t.Fatalf("expected menu item availability unavailable, got %q", got.Menu.Items[0].Availability)
	}
}
