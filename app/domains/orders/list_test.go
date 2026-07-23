package orders_test

import (
	"fmt"
	"testing"
	"time"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestOrders_ListExpressionFilters(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()

	base := b.WithIngredient("Tequila", measurement.UnitOz)
	drink := b.WithDrink(drinksmodels.Drink{
		Name: "Paloma", Category: drinksmodels.DrinkCategoryHighball, Glass: drinksmodels.GlassTypeHighball,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: base.ID, Amount: measurement.MustAmount(2, measurement.UnitOz)}},
			Steps:       []string{"Build"},
		},
	})
	targetMenu := b.WithPublishedMenu(menumodels.Menu{Name: "Priority Menu"}, drink)
	decoyMenu := b.WithPublishedMenu(menumodels.Menu{Name: "Walk-in Menu"}, drink)
	target := b.WithOrder(ordersmodels.Order{
		MenuID: targetMenu.ID, Notes: "priority table",
		Items: []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
	})
	b.WithOrder(ordersmodels.Order{
		MenuID: decoyMenu.ID, Notes: "walk-in table",
		Items: []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
	})
	var err error
	target, err = f.Orders.Cancel(f.OwnerContext(), &ordersmodels.Order{ID: target.ID})
	testutil.Ok(t, err)

	tests := map[string]string{
		"id":         fmt.Sprintf("id == %q", target.ID.String()),
		"menu_id":    fmt.Sprintf("menu_id == %q", target.MenuID.String()),
		"status":     `status == "cancelled"`,
		"created_at": fmt.Sprintf("created_at == date(%q)", target.CreatedAt.Format(time.RFC3339Nano)),
		"notes":      `notes.contains("priority")`,
	}
	for name, expression := range tests {
		ctx := f.ActorContext("owner")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			page, err := f.Orders.List(ctx, orders.ListRequest{Filter: expression})
			testutil.Ok(t, err)
			testutil.Equals(t, len(page.Items), 1)
			testutil.Equals(t, page.Items[0].ID, target.ID)
		})
	}
}
