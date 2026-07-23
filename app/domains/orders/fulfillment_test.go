package orders_test

import (
	"context"
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	ordersauthz "github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestCompleteOrderUsesCatalogRatioForExplicitSubstitute(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	primary := b.WithIngredientModel(ingredientsmodels.Ingredient{
		Name: "Simple Syrup", Category: ingredientsmodels.CategorySyrup, Unit: measurement.UnitOz,
	})
	substitute := b.WithIngredientModel(ingredientsmodels.Ingredient{
		Name: "Honey Syrup", Category: ingredientsmodels.CategorySyrup, Unit: measurement.UnitOz,
	})
	substituteStock := b.WithInventory(substitute, 3)
	drink := b.WithDrink(drinksmodels.Drink{
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
	menu := b.WithPublishedMenu(menumodels.Menu{Name: "Substitution Menu"}, drink)
	testutil.Equals(t, orderMenuAvailability(menu, drink.ID), menumodels.AvailabilityLimited)
	order := b.WithOrder(ordersmodels.Order{
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
	testutil.Equals(t, remaining.Amount, measurement.MustAmount(0, substitute.Unit), testutil.EquateAmounts(0.000001))
	gotMenu, err := f.Menus.Get(ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, orderMenuAvailability(gotMenu, drink.ID), menumodels.AvailabilityUnavailable)

	entry := f.LatestAuditEntry(ordersauthz.ActionComplete)
	testutil.AuditTouches(t, entry, order.ID.EntityUID(), substituteStock.EntityUID(), menu.ID.EntityUID())
}

func TestCompleteOrderPrefersHigherQualityCatalogSubstitute(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	primary := b.WithIngredientModel(ingredientsmodels.Ingredient{
		Name: "Bourbon", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz,
	})
	rye := b.WithIngredientModel(ingredientsmodels.Ingredient{
		Name: "Rye Whiskey", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz,
	})
	scotch := b.WithIngredientModel(ingredientsmodels.Ingredient{
		Name: "Scotch", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz,
	})
	ryeStock := b.WithInventory(rye, 5)
	scotchStock := b.WithInventory(scotch, 10)
	drink := b.WithDrink(drinksmodels.Drink{
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
	menu := b.WithPublishedMenu(menumodels.Menu{Name: "Quality Menu"}, drink)
	testutil.Equals(t, orderMenuAvailability(menu, drink.ID), menumodels.AvailabilityAvailable)
	order := b.WithOrder(ordersmodels.Order{
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
	testutil.Equals(t, remainingRye, &wantRye, testutil.EquateAmounts(0.000001))
	remainingScotch, err := f.Inventory.Get(ctx, scotch.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, remainingScotch, scotchStock, testutil.EquateAmounts(0.000001))

	entry := f.LatestAuditEntry(ordersauthz.ActionComplete)
	testutil.AuditTouches(t, entry, order.ID.EntityUID(), ryeStock.EntityUID())
}

func TestCompleteOrderBacktracksWhenPreferredSubstituteIsShared(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	first := b.WithIngredient("Reservation First", measurement.UnitOz)
	second := b.WithIngredient("Reservation Second", measurement.UnitOz)
	shared := b.WithIngredient("Reservation Shared", measurement.UnitOz)
	fallback := b.WithIngredient("Reservation Fallback", measurement.UnitOz)
	b.WithInventory(shared, 1.5)
	b.WithInventory(fallback, 1)
	drink := b.WithDrink(drinksmodels.Drink{
		Name: "Reservation Cocktail", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{IngredientID: first.ID, Amount: measurement.MustAmount(1, measurement.UnitOz), Substitutes: []entity.IngredientID{shared.ID}},
				{IngredientID: second.ID, Amount: measurement.MustAmount(1, measurement.UnitOz), Substitutes: []entity.IngredientID{shared.ID, fallback.ID}},
			},
			Steps: []string{"Shake"},
		},
	})
	menu := b.WithPublishedMenu(menumodels.Menu{Name: "Reservation Menu"}, drink)
	order := b.WithOrder(ordersmodels.Order{
		MenuID: menu.ID,
		Items:  []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
	})

	_, err := f.Orders.Complete(ctx, &ordersmodels.Order{ID: order.ID})
	testutil.Ok(t, err)
	remainingShared, err := f.Inventory.Get(ctx, shared.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, remainingShared.Amount, measurement.MustAmount(0.5, measurement.UnitOz), testutil.EquateAmounts(0.000001))
	remainingFallback, err := f.Inventory.Get(ctx, fallback.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, remainingFallback.Amount, measurement.MustAmount(0, measurement.UnitOz), testutil.EquateAmounts(0.000001))
}

func TestMenuAvailabilityReservesSharedSubstitute(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()

	first := b.WithIngredient("Unavailable First", measurement.UnitOz)
	second := b.WithIngredient("Unavailable Second", measurement.UnitOz)
	shared := b.WithIngredient("Unavailable Shared", measurement.UnitOz)
	b.WithInventory(shared, 1.5)
	drink := b.WithDrink(drinksmodels.Drink{
		Name: "Unavailable Shared Cocktail", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{IngredientID: first.ID, Amount: measurement.MustAmount(1, measurement.UnitOz), Substitutes: []entity.IngredientID{shared.ID}},
				{IngredientID: second.ID, Amount: measurement.MustAmount(1, measurement.UnitOz), Substitutes: []entity.IngredientID{shared.ID}},
			},
			Steps: []string{"Shake"},
		},
	})

	menu := b.WithPublishedMenu(menumodels.Menu{Name: "Unavailable Shared Menu"}, drink)
	testutil.Equals(t, orderMenuAvailability(menu, drink.ID), menumodels.AvailabilityUnavailable)
}

func TestCompleteOrderPreservesFulfillmentDependencyError(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	ingredient := b.WithIngredient("Cancellation Ingredient", measurement.UnitOz)
	b.WithInventory(ingredient, 10)
	drink := b.WithDrink(drinksmodels.Drink{
		Name: "Cancellation Cocktail", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
			Steps:       []string{"Shake"},
		},
	})
	menu := b.WithPublishedMenu(menumodels.Menu{Name: "Cancellation Menu"}, drink)
	order := b.WithOrder(ordersmodels.Order{
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

func orderMenuAvailability(menu *menumodels.Menu, drinkID entity.DrinkID) menumodels.Availability {
	for _, item := range menu.Items {
		if item.DrinkID == drinkID {
			return item.Availability
		}
	}
	return ""
}
