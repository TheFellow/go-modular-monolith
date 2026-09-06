package handlers_test

import (
	dm "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	im "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	iv "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	om "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"testing"
)

func TestMissingReservationPreventsCompletionAndRollsBackOrder(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()
	i := testutil.CreateIngredient(t, f, im.Ingredient{Name: "Reserved", Category: im.CategoryOther, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, iv.Update{IngredientID: i.ID, Amount: measurement.MustAmount(5, i.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	d := testutil.CreateDrink(t, f, dm.Drink{Name: "Recipe", Recipe: dm.Recipe{Ingredients: []dm.RecipeIngredient{{IngredientID: i.ID, Amount: measurement.MustAmount(2, i.Unit)}}, Steps: []string{"Mix"}}})
	m := testutil.CreateMenu(t, f, "Menu", testutil.WithDrink(d), testutil.Published())
	order := testutil.PlaceOrder(t, f, om.Order{MenuID: m.ID, Items: []om.OrderItem{{DrinkID: d.ID, Quantity: 1}}})
	inventory := dao.New(f.Store, tagging.NewRepository(f.Store))
	testutil.Ok(t, f.Store.Write(ctx, func(tx *store.Tx) error { return inventory.DeleteReservations(ctx.WithTransaction(tx), order.ID) }))
	_, err := f.Orders.Complete(ctx, order)
	testutil.ErrorIsFailedPrecondition(t, err)
	current, err := f.Orders.Get(ctx, order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, current.Status, om.OrderStatusPending)
	testutil.Equals(t, current.Revision, order.Revision)
	stock, err := f.Inventory.Get(ctx, i.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Amount.Value(), 5.0)
}
