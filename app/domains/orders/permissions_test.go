package orders_test

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPermissions_Orders(t *testing.T) {
	fix := testutil.NewFixture(t)
	a := fix.App

	owner := fix.Ctx
	anon := fix.AsActor("anonymous")

	t.Run("owner", func(t *testing.T) {
		_, err := a.Orders.List(owner, orders.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Orders.Get(owner, ordersmodels.NewOrderID("does-not-exist"))
		testutil.RequireNotDenied(t, err)

		_, err = a.Orders.Place(owner, ordersmodels.Order{
			ID:     ordersmodels.NewOrderID(""),
			MenuID: menumodels.NewMenuID("does-not-exist"),
			Items: []ordersmodels.OrderItem{
				{DrinkID: drinksmodels.NewDrinkID("does-not-exist"), Quantity: 1},
			},
		})
		testutil.RequireNotDenied(t, err)

		_, err = a.Orders.Complete(owner, ordersmodels.Order{ID: ordersmodels.NewOrderID("does-not-exist")})
		testutil.RequireNotDenied(t, err)

		_, err = a.Orders.Cancel(owner, ordersmodels.Order{ID: ordersmodels.NewOrderID("does-not-exist")})
		testutil.RequireNotDenied(t, err)
	})

	t.Run("anonymous", func(t *testing.T) {
		_, err := a.Orders.List(anon, orders.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Orders.Get(anon, ordersmodels.NewOrderID("does-not-exist"))
		testutil.RequireNotDenied(t, err)

		_, err = a.Orders.Place(anon, ordersmodels.Order{
			ID:     ordersmodels.NewOrderID(""),
			MenuID: menumodels.NewMenuID("does-not-exist"),
			Items: []ordersmodels.OrderItem{
				{DrinkID: drinksmodels.NewDrinkID("does-not-exist"), Quantity: 1},
			},
		})
		testutil.RequireDenied(t, err)

		_, err = a.Orders.Complete(anon, ordersmodels.Order{ID: ordersmodels.NewOrderID("does-not-exist")})
		testutil.RequireDenied(t, err)

		_, err = a.Orders.Cancel(anon, ordersmodels.Order{ID: ordersmodels.NewOrderID("does-not-exist")})
		testutil.RequireDenied(t, err)
	})
}
