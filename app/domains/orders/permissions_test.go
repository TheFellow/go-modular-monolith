package orders_test

import (
	"testing"

	drinksM "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menuM "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	ordersM "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPermissions_Orders(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		canRead   bool
		canManage bool
	}{
		{name: "owner", canRead: true, canManage: true},
		{name: "manager", canRead: true, canManage: true},
		{name: "sommelier", canRead: true, canManage: false},
		{name: "bartender", canRead: true, canManage: true},
		{name: "anonymous", canRead: false, canManage: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := testutil.NewFixture(t)
			b := f.Bootstrap()
			a := f.App
			owner := f.OwnerContext()
			var ctx *middleware.Context
			if tc.name == "owner" {
				ctx = owner
			} else {
				ctx = f.ActorContext(tc.name)
			}

			base := b.WithIngredient("Orders Permissions Base", measurement.UnitOz)
			b.WithInventory(base, 100)
			drink := b.WithDrink(drinksM.Drink{
				Name:     "Order Drink",
				Category: drinksM.DrinkCategoryCocktail,
				Glass:    drinksM.GlassTypeCoupe,
				Recipe: drinksM.Recipe{
					Ingredients: []drinksM.RecipeIngredient{
						{IngredientID: base.ID, Amount: measurement.MustAmount(1.0, measurement.UnitOz)},
					},
					Steps: []string{"Shake"},
				},
			})
			menu := b.WithMenu("Orders Menu")
			menu, err := a.Menus.AddDrink(f.OwnerContext(), &menuM.MenuPatch{MenuID: menu.ID, DrinkID: drink.ID})
			testutil.Ok(t, err)
			menu, err = a.Menus.Publish(f.OwnerContext(), &menuM.Menu{ID: menu.ID})
			testutil.Ok(t, err)

			readOrder := b.WithOrder(ordersM.Order{
				MenuID: menu.ID,
				Items:  []ordersM.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
			})
			completeOrder := b.WithOrder(ordersM.Order{
				MenuID: menu.ID,
				Items:  []ordersM.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
			})
			cancelOrder := b.WithOrder(ordersM.Order{
				MenuID: menu.ID,
				Items:  []ordersM.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
			})

			listed, err := a.Orders.List(ctx, orders.ListRequest{})
			testutil.Ok(t, err)
			wantCount := 0
			if tc.canRead {
				wantCount = 3
			}
			testutil.ErrorIf(t, len(listed.Items) != wantCount, "expected %d visible orders, got %d", wantCount, len(listed.Items))

			_, err = a.Orders.Get(ctx, readOrder.ID)
			if tc.canRead {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}

			_, err = a.Orders.Place(ctx, &ordersM.Order{
				MenuID: menu.ID,
				Items:  []ordersM.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
			})
			if tc.canManage {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}

			_, err = a.Orders.Complete(ctx, &ordersM.Order{ID: completeOrder.ID})
			if tc.canManage {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}

			_, err = a.Orders.Cancel(ctx, &ordersM.Order{ID: cancelOrder.ID})
			if tc.canManage {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}

			persistedCount, err := a.Orders.Count(owner, orders.ListRequest{})
			testutil.Ok(t, err)
			wantPersistedCount := 3
			if tc.canManage {
				wantPersistedCount++
			}
			testutil.Equals(t, persistedCount, wantPersistedCount)
			gotComplete, err := a.Orders.Get(owner, completeOrder.ID)
			testutil.Ok(t, err)
			wantCompleteStatus := ordersM.OrderStatusPending
			if tc.canManage {
				wantCompleteStatus = ordersM.OrderStatusCompleted
			}
			testutil.Equals(t, gotComplete.Status, wantCompleteStatus)
			gotCancel, err := a.Orders.Get(owner, cancelOrder.ID)
			testutil.Ok(t, err)
			wantCancelStatus := ordersM.OrderStatusPending
			if tc.canManage {
				wantCancelStatus = ordersM.OrderStatusCancelled
			}
			testutil.Equals(t, gotCancel.Status, wantCancelStatus)
		})
	}
}
