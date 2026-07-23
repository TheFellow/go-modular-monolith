package handlers_test

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	ordersauthz "github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestOrderCompletedHandlersDepleteUsedStockAndPreserveUnrelatedStock(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	used := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Used", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	other := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Other", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	usedStock := testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: used.ID, Amount: measurement.MustAmount(2, used.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	otherStock := testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: other.ID, Amount: measurement.MustAmount(10, other.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	affectedDrink := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "Affected", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: used.ID, Amount: measurement.MustAmount(2, used.Unit)}}, Steps: []string{"Mix"}},
	})
	survivor := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "Survivor", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: other.ID, Amount: measurement.MustAmount(1, other.Unit)}}, Steps: []string{"Mix"}},
	})
	affectedMenu := testutil.CreateMenu(t, f, "Affected", testutil.WithDrink(affectedDrink), testutil.WithDrink(survivor), testutil.Published())
	unrelatedMenu := testutil.CreateMenu(t, f, "Unrelated", testutil.WithDrink(survivor), testutil.Published())
	order := testutil.PlaceOrder(t, f, ordersmodels.Order{
		MenuID: affectedMenu.ID,
		Items:  []ordersmodels.OrderItem{{DrinkID: affectedDrink.ID, Quantity: 1}},
	})

	completed, err := f.Orders.Complete(ctx, &ordersmodels.Order{ID: order.ID})
	testutil.Ok(t, err)
	testutil.Equals(t, completed.Status, ordersmodels.OrderStatusCompleted)
	gotUsedStock, err := f.Inventory.Get(ctx, used.ID)
	testutil.Ok(t, err)
	wantUsedStock := *usedStock
	wantUsedStock.Amount = measurement.MustAmount(0, used.Unit)
	wantUsedStock.LastUpdated = gotUsedStock.LastUpdated
	testutil.Equals(t, gotUsedStock, &wantUsedStock)
	gotOtherStock, err := f.Inventory.Get(ctx, other.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotOtherStock, otherStock)
	gotMenu, err := f.Menus.Get(ctx, affectedMenu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, menuAvailability(gotMenu, affectedDrink.ID), menumodels.AvailabilityUnavailable)
	testutil.Equals(t, menuAvailability(gotMenu, survivor.ID), menumodels.AvailabilityAvailable)
	gotUnrelatedMenu, err := f.Menus.Get(ctx, unrelatedMenu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotUnrelatedMenu, unrelatedMenu, cmpopts.EquateEmpty())

	entry := f.LatestAuditEntry(ordersauthz.ActionComplete)
	testutil.AuditTouches(t, entry, order.ID.EntityUID(), usedStock.EntityUID(), affectedMenu.ID.EntityUID())
}

func menuAvailability(menu *menumodels.Menu, drinkID entity.DrinkID) menumodels.Availability {
	for _, item := range menu.Items {
		if item.DrinkID == drinkID {
			return item.Availability
		}
	}
	return ""
}
