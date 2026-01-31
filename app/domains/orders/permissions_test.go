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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := testutil.NewFixture(t)
			b := f.Bootstrap()
			a := f.App
			var ctx *middleware.Context
			if tc.name == "owner" {
				ctx = f.OwnerContext()
			} else {
				ctx = f.ActorContext(tc.name)
			}

			base := b.WithIngredient("Orders Permissions Base", measurement.UnitOz)
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
			order, err := a.Orders.Place(f.OwnerContext(), &ordersM.Order{
				MenuID: menu.ID,
				Items: []ordersM.OrderItem{
					{DrinkID: drink.ID, Quantity: 1},
				},
			})
			testutil.Ok(t, err)

			_, err = a.Orders.List(ctx, orders.ListRequest{})
			if tc.canRead {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			_, err = a.Orders.Get(ctx, ordersM.NewOrderID("does-not-exist"))
			if tc.canRead {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			_, err = a.Orders.Place(ctx, &ordersM.Order{
				ID:     ordersM.NewOrderID(""),
				MenuID: menuM.NewMenuID("does-not-exist"),
				Items: []ordersM.OrderItem{
					{DrinkID: drinksM.NewDrinkID("does-not-exist"), Quantity: 1},
				},
			})
			if tc.canManage {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			_, err = a.Orders.Complete(ctx, &ordersM.Order{ID: order.ID})
			if tc.canManage {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			_, err = a.Orders.Cancel(ctx, &ordersM.Order{ID: order.ID})
			if tc.canManage {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}
		})
	}
}
