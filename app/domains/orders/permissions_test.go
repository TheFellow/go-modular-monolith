package orders_test

import (
	"testing"

	drinksM "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menuM "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	ordersM "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPermissions_Orders(t *testing.T) {
	t.Parallel()

	t.Run("owner", func(t *testing.T) {
		t.Parallel()
		f := testutil.NewFixture(t)
		a := f.App
		owner := f.OwnerContext()

		_, err := a.Orders.List(owner, orders.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Orders.Get(owner, ordersM.NewOrderID("does-not-exist"))
		testutil.RequireNotDenied(t, err)

		_, err = a.Orders.Place(owner, ordersM.Order{
			ID:     ordersM.NewOrderID(""),
			MenuID: menuM.NewMenuID("does-not-exist"),
			Items: []ordersM.OrderItem{
				{DrinkID: drinksM.NewDrinkID("does-not-exist"), Quantity: 1},
			},
		})
		testutil.RequireNotDenied(t, err)

		_, err = a.Orders.Complete(owner, ordersM.Order{ID: ordersM.NewOrderID("does-not-exist")})
		testutil.RequireNotDenied(t, err)

		_, err = a.Orders.Cancel(owner, ordersM.Order{ID: ordersM.NewOrderID("does-not-exist")})
		testutil.RequireNotDenied(t, err)
	})

	t.Run("anonymous", func(t *testing.T) {
		t.Parallel()
		f := testutil.NewFixture(t)
		a := f.App
		anon := f.ActorContext("anonymous")

		_, err := a.Orders.List(anon, orders.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Orders.Get(anon, ordersM.NewOrderID("does-not-exist"))
		testutil.RequireNotDenied(t, err)

		_, err = a.Orders.Place(anon, ordersM.Order{
			ID:     ordersM.NewOrderID(""),
			MenuID: menuM.NewMenuID("does-not-exist"),
			Items: []ordersM.OrderItem{
				{DrinkID: drinksM.NewDrinkID("does-not-exist"), Quantity: 1},
			},
		})
		testutil.RequireDenied(t, err)

		_, err = a.Orders.Complete(anon, ordersM.Order{ID: ordersM.NewOrderID("does-not-exist")})
		testutil.RequireDenied(t, err)

		_, err = a.Orders.Cancel(anon, ordersM.Order{ID: ordersM.NewOrderID("does-not-exist")})
		testutil.RequireDenied(t, err)
	})
}
