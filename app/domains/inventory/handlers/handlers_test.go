package handlers_test

import (
	"testing"

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

	used := b.WithIngredient("Used", measurement.UnitOz)
	other := b.WithIngredient("Other", measurement.UnitOz)
	usedStock := b.WithInventory(used, 2)
	otherStock := b.WithInventory(other, 10)
	affectedDrink := f.CreateDrink("Affected").WithIngredient(used, 2).Build()
	survivor := f.CreateDrink("Survivor").WithIngredient(other, 1).Build()
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

func menuAvailability(menu *menumodels.Menu, drinkID entity.DrinkID) menumodels.Availability {
	for _, item := range menu.Items {
		if item.DrinkID == drinkID {
			return item.Availability
		}
	}
	return ""
}
