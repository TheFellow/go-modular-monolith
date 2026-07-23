package orders_test

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestOrders_PlaceGetCancelAndComplete(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	base := b.WithIngredient("Order Base", measurement.UnitOz)
	initialStock := b.WithInventory(base, 10)
	drink := b.WithDrink(drinksmodels.Drink{
		Name: "Service Cocktail", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: base.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
			Steps:       []string{"Shake"},
		},
	})
	menu := b.WithPublishedMenu(menumodels.Menu{Name: "Service Menu"}, drink)

	count, err := f.Orders.Count(ctx, orders.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, count, 0)

	cancelledOrder := b.WithOrder(models.Order{
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

	completedOrder := b.WithOrder(models.Order{
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
	testutil.Equals(t, stock, &wantStock, testutil.EquateAmounts(0.000001))
	count, err = f.Orders.Count(ctx, orders.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, count, 2)
}
