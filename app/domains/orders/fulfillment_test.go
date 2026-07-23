package orders_test

import (
	"context"
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
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestCompleteOrderUsesCatalogRatioForExplicitSubstitute(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	primary := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Simple Syrup", Category: ingredientsmodels.CategorySyrup, Unit: measurement.UnitOz,
	})
	substitute := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Honey Syrup", Category: ingredientsmodels.CategorySyrup, Unit: measurement.UnitOz,
	})
	substituteStock := testutil.SetInventory(t, f, fulfillmentStock(substitute, 3))
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "Honey Sour", Category: drinksmodels.DrinkCategorySour, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{
				IngredientID: primary.ID,
				Amount:       measurement.MustAmount(2, measurement.UnitOz),
				Substitutes:  []entity.IngredientID{substitute.ID},
			}},
			Steps: []string{"Shake"},
		},
	})
	menu := testutil.CreateMenu(t, f, "Substitution Menu", testutil.WithDrink(drink), testutil.Published())
	testutil.Equals(t, orderMenuAvailability(menu, drink.ID), menumodels.AvailabilityLimited)
	order := testutil.PlaceOrder(t, f, ordersmodels.Order{
		MenuID: menu.ID,
		Items:  []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 2}},
	})

	completed, err := f.Orders.Complete(ctx, &ordersmodels.Order{ID: order.ID})
	testutil.Ok(t, err)
	testutil.Equals(t, completed.Status, ordersmodels.OrderStatusCompleted)
	testutil.IsTrue(t, completed.CompletedAt.IsSome())

	_, err = f.Inventory.Get(ctx, primary.ID)
	testutil.ErrorIsNotFound(t, err)
	remaining, err := f.Inventory.Get(ctx, substitute.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, remaining.Amount, measurement.MustAmount(0, substitute.Unit))
	gotMenu, err := f.Menus.Get(ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, orderMenuAvailability(gotMenu, drink.ID), menumodels.AvailabilityUnavailable)

	entry := f.LatestAuditEntry(ordersauthz.ActionComplete)
	testutil.AuditTouches(t, entry, order.ID.EntityUID(), substituteStock.EntityUID(), menu.ID.EntityUID())
}

func TestCompleteOrderPrefersHigherQualityCatalogSubstitute(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	primary := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Bourbon", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz,
	})
	rye := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Rye Whiskey", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz,
	})
	scotch := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Scotch", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz,
	})
	ryeStock := testutil.SetInventory(t, f, fulfillmentStock(rye, 5))
	scotchStock := testutil.SetInventory(t, f, fulfillmentStock(scotch, 10))
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "Whiskey Cocktail", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeRocks,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{
				IngredientID: primary.ID,
				Amount:       measurement.MustAmount(1, measurement.UnitOz),
				Substitutes:  []entity.IngredientID{scotch.ID},
			}},
			Steps: []string{"Stir"},
		},
	})
	menu := testutil.CreateMenu(t, f, "Quality Menu", testutil.WithDrink(drink), testutil.Published())
	testutil.Equals(t, orderMenuAvailability(menu, drink.ID), menumodels.AvailabilityAvailable)
	order := testutil.PlaceOrder(t, f, ordersmodels.Order{
		MenuID: menu.ID,
		Items:  []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 2}},
	})

	_, err := f.Orders.Complete(ctx, &ordersmodels.Order{ID: order.ID})
	testutil.Ok(t, err)
	remainingRye, err := f.Inventory.Get(ctx, rye.ID)
	testutil.Ok(t, err)
	wantRye := *ryeStock
	wantRye.Amount = measurement.MustAmount(3, rye.Unit)
	wantRye.LastUpdated = remainingRye.LastUpdated
	testutil.Equals(t, remainingRye, &wantRye)
	remainingScotch, err := f.Inventory.Get(ctx, scotch.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, remainingScotch, scotchStock)

	entry := f.LatestAuditEntry(ordersauthz.ActionComplete)
	testutil.AuditTouches(t, entry, order.ID.EntityUID(), ryeStock.EntityUID())
}

func TestCompleteOrderBacktracksWhenPreferredSubstituteIsShared(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	first := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Reservation First", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	second := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Reservation Second", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	shared := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Reservation Shared", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	fallback := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Reservation Fallback", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, fulfillmentStock(shared, 1.5))
	testutil.SetInventory(t, f, fulfillmentStock(fallback, 1))
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "Reservation Cocktail", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{IngredientID: first.ID, Amount: measurement.MustAmount(1, measurement.UnitOz), Substitutes: []entity.IngredientID{shared.ID}},
				{IngredientID: second.ID, Amount: measurement.MustAmount(1, measurement.UnitOz), Substitutes: []entity.IngredientID{shared.ID, fallback.ID}},
			},
			Steps: []string{"Shake"},
		},
	})
	menu := testutil.CreateMenu(t, f, "Reservation Menu", testutil.WithDrink(drink), testutil.Published())
	order := testutil.PlaceOrder(t, f, ordersmodels.Order{
		MenuID: menu.ID,
		Items:  []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
	})

	_, err := f.Orders.Complete(ctx, &ordersmodels.Order{ID: order.ID})
	testutil.Ok(t, err)
	remainingShared, err := f.Inventory.Get(ctx, shared.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, remainingShared.Amount, measurement.MustAmount(0.5, measurement.UnitOz))
	remainingFallback, err := f.Inventory.Get(ctx, fallback.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, remainingFallback.Amount, measurement.MustAmount(0, measurement.UnitOz))
}

func TestMenuAvailabilityReservesSharedSubstitute(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	first := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Unavailable First", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	second := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Unavailable Second", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	shared := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Unavailable Shared", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, fulfillmentStock(shared, 1.5))
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "Unavailable Shared Cocktail", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{IngredientID: first.ID, Amount: measurement.MustAmount(1, measurement.UnitOz), Substitutes: []entity.IngredientID{shared.ID}},
				{IngredientID: second.ID, Amount: measurement.MustAmount(1, measurement.UnitOz), Substitutes: []entity.IngredientID{shared.ID}},
			},
			Steps: []string{"Shake"},
		},
	})

	menu := testutil.CreateMenu(t, f, "Unavailable Shared Menu", testutil.WithDrink(drink), testutil.Published())
	testutil.Equals(t, orderMenuAvailability(menu, drink.ID), menumodels.AvailabilityUnavailable)
}

func TestCompleteOrderPreservesFulfillmentDependencyError(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Cancellation Ingredient", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, fulfillmentStock(ingredient, 10))
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "Cancellation Cocktail", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
			Steps:       []string{"Shake"},
		},
	})
	menu := testutil.CreateMenu(t, f, "Cancellation Menu", testutil.WithDrink(drink), testutil.Published())
	order := testutil.PlaceOrder(t, f, ordersmodels.Order{
		MenuID: menu.ID,
		Items:  []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
	})
	tx, err := f.Store.Begin(ctx, true)
	testutil.Ok(t, err)
	t.Cleanup(func() { testutil.Ok(t, f.Store.Rollback(tx)) })
	cancelledParent, cancel := context.WithCancel(ctx)
	cancel()
	cancelledCtx := middleware.NewContext(cancelledParent).WithTransaction(tx)

	_, err = f.Orders.Complete(cancelledCtx, &ordersmodels.Order{ID: order.ID})
	testutil.ErrorIs(t, err, context.Canceled)
}

func fulfillmentStock(ingredient *ingredientsmodels.Ingredient, quantity float64) inventorymodels.Update {
	return inventorymodels.Update{
		IngredientID: ingredient.ID,
		Amount:       measurement.MustAmount(quantity, ingredient.Unit),
		CostPerUnit:  money.NewPriceFromCents(100, currency.USD),
	}
}

func orderMenuAvailability(menu *menumodels.Menu, drinkID entity.DrinkID) menumodels.Availability {
	for _, item := range menu.Items {
		if item.DrinkID == drinkID {
			return item.Availability
		}
	}
	return ""
}
