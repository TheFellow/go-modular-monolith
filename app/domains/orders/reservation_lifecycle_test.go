package orders_test

import (
	"math"
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	ordersauthz "github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestReservationShortageBlocksRestockUnblocksAndCancellationReleases(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Reserved lime", Category: ingredientsmodels.CategoryJuice, Unit: measurement.UnitOz})
	stock := testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(10, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{Name: "Reserved daiquiri", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe, Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(2, ingredient.Unit)}}, Steps: []string{"Shake"}}})
	menu := testutil.CreateMenu(t, f, "Reservation menu", testutil.WithDrink(drink), testutil.Published())
	order := testutil.PlaceOrder(t, f, ordersmodels.Order{MenuID: menu.ID, Items: []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 2}}})

	reserved, err := f.Inventory.Get(ctx, ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, reserved.Amount.Value(), 10.0)
	testutil.Equals(t, reserved.ReservedAmount().Value(), 4.0)
	testutil.IsTrue(t, math.Abs(reserved.Available().Value()-6.0) < 1e-9)
	testutil.Equals(t, order.IngredientUsage[0].Amount.Value(), 4.0)

	_, err = f.Inventory.Set(ctx, &inventorymodels.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(2, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	testutil.Ok(t, err)
	blocked, err := f.Orders.Get(ctx, order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, blocked.Status, ordersmodels.OrderStatusBlocked)
	testutil.Equals(t, blocked.BlockedIngredients, []entity.IngredientID{ingredient.ID})

	_, err = f.Orders.Complete(ctx, &ordersmodels.Order{ID: order.ID})
	testutil.ErrorIsInvalid(t, err)
	_, err = f.Inventory.Set(ctx, &inventorymodels.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(10, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	testutil.Ok(t, err)
	unblocked, err := f.Orders.Get(ctx, order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, unblocked.Status, ordersmodels.OrderStatusPending)

	_, err = f.Orders.Cancel(ctx, &ordersmodels.Order{ID: order.ID})
	testutil.Ok(t, err)
	released, err := f.Inventory.Get(ctx, ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, released.ReservedAmount().Value(), 0.0)
	testutil.AuditTouches(t, f.LatestAuditEntry(ordersauthz.ActionCancel), order.ID.EntityUID(), stock.EntityUID())
}

func TestIngredientRetirementBlocksReservedOrderButStillAllowsCancellation(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Retired reserve", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(5, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{Name: "Retired recipe", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe, Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, ingredient.Unit)}}, Steps: []string{"Stir"}}})
	menu := testutil.CreateMenu(t, f, "Retired reservation", testutil.WithDrink(drink), testutil.Published())
	order := testutil.PlaceOrder(t, f, ordersmodels.Order{MenuID: menu.ID, Items: []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}}})

	_, err := f.Ingredients.Delete(ctx, ingredient.ID)
	testutil.Ok(t, err)
	blocked, err := f.Orders.Get(ctx, order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, blocked.Status, ordersmodels.OrderStatusBlocked)
	_, err = f.Orders.Cancel(ctx, &ordersmodels.Order{ID: order.ID})
	testutil.Ok(t, err)
	cancelled, err := f.Orders.Get(ctx, order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, cancelled.Status, ordersmodels.OrderStatusCancelled)
}
