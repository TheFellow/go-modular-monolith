package handlers_test

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	ordersauthz "github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestOrderCompletedHandlersDepleteUsedStockAndPreserveUnrelatedStock(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	used := b.WithIngredientModel(ingredientsmodels.Ingredient{Name: "Used", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	other := b.WithIngredientModel(ingredientsmodels.Ingredient{Name: "Other", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	usedStock := b.WithInventory(used, 2)
	otherStock := b.WithInventory(other, 10)
	affectedDrink := inventoryHandlerDrink(b, "Affected", used, 2)
	survivor := inventoryHandlerDrink(b, "Survivor", other, 1)
	affectedMenu := b.WithPublishedMenu(menumodels.Menu{Name: "Affected"}, affectedDrink, survivor)
	unrelatedMenu := b.WithPublishedMenu(menumodels.Menu{Name: "Unrelated"}, survivor)
	order := b.WithOrder(ordersmodels.Order{
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
	testutil.Equals(t, gotUsedStock, &wantUsedStock, testutil.EquateAmounts(0.000001))
	gotOtherStock, err := f.Inventory.Get(ctx, other.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotOtherStock, otherStock, testutil.EquateAmounts(0.000001))
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

func inventoryHandlerDrink(b *testutil.Bootstrap, name string, ingredient *ingredientsmodels.Ingredient, amount float64) *drinksmodels.Drink {
	return b.WithDrink(drinksmodels.Drink{
		Name: name, Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(amount, ingredient.Unit)}},
			Steps:       []string{"Mix"},
		},
	})
}

func menuAvailability(menu *menumodels.Menu, drinkID entity.DrinkID) menumodels.Availability {
	for _, item := range menu.Items {
		if item.DrinkID == drinkID {
			return item.Availability
		}
	}
	return ""
}
