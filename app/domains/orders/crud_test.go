package orders_test

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestOrders_PlaceGetCancelAndComplete(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	base := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Order Base", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
	})
	initialStock := testutil.SetInventory(t, f, inventorymodels.Update{
		IngredientID: base.ID, Amount: measurement.MustAmount(10, base.Unit),
		CostPerUnit: money.NewPriceFromCents(100, currency.USD),
	})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "Service Cocktail", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: base.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
			Steps:       []string{"Shake"},
		},
	})
	menu := testutil.CreateMenu(t, f, "Service Menu", testutil.WithDrink(drink), testutil.Published())

	count, err := f.Orders.Count(ctx, orders.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, count, 0)

	cancelledOrder := testutil.PlaceOrder(t, f, models.Order{
		MenuID: menu.ID, Notes: "cancel me",
		Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
	})
	got, err := f.Orders.Get(ctx, cancelledOrder.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, cancelledOrder)
	wantCancelled := *cancelledOrder
	wantCancelled.Status = models.OrderStatusCancelled
	cancelledOrder, err = f.Orders.Cancel(ctx, &models.Order{ID: cancelledOrder.ID})
	testutil.Ok(t, err)
	testutil.Equals(t, cancelledOrder, &wantCancelled)

	completedOrder := testutil.PlaceOrder(t, f, models.Order{
		MenuID: menu.ID, Notes: "complete me",
		Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 2}},
	})
	wantCompleted := *completedOrder
	wantCompleted.Status = models.OrderStatusCompleted
	completedOrder, err = f.Orders.Complete(ctx, &models.Order{ID: completedOrder.ID})
	testutil.Ok(t, err)
	wantCompleted.CompletedAt = completedOrder.CompletedAt
	testutil.Equals(t, completedOrder, &wantCompleted)
	got, err = f.Orders.Get(ctx, completedOrder.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, completedOrder)

	stock, err := f.Inventory.Get(ctx, base.ID)
	testutil.Ok(t, err)
	wantStock := *initialStock
	wantStock.Amount = measurement.MustAmount(8, base.Unit)
	wantStock.LastUpdated = stock.LastUpdated
	testutil.Equals(t, stock, &wantStock)
	count, err = f.Orders.Count(ctx, orders.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, count, 2)
}
