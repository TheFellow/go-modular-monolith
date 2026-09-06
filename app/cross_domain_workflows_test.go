package app_test

import (
	dm "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	im "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	iv "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	mm "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	om "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"reflect"
	"testing"
)

func workflowFixture(t *testing.T) (*testutil.Fixture, *im.Ingredient, *dm.Drink, *mm.Menu) {
	t.Helper()
	f := testutil.NewFixture(t)
	i := testutil.CreateIngredient(t, f, im.Ingredient{Name: "Original", Category: im.CategorySpirit, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, workflowStock(i, 10))
	d := testutil.CreateDrink(t, f, dm.Drink{Name: "Original cocktail", Recipe: dm.Recipe{Ingredients: []dm.RecipeIngredient{{IngredientID: i.ID, Amount: measurement.MustAmount(2, i.Unit)}}, Steps: []string{"Original instructions"}}})
	m := testutil.CreateMenu(t, f, "Original menu", testutil.WithDrink(d), testutil.Published())
	return f, i, d, m
}
func workflowStock(i *im.Ingredient, n float64) iv.Update {
	return iv.Update{IngredientID: i.ID, Amount: measurement.MustAmount(n, i.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)}
}
func workflowOrder(t *testing.T, f *testutil.Fixture, d *dm.Drink, m *mm.Menu) *om.Order {
	t.Helper()
	return testutil.PlaceOrder(t, f, om.Order{MenuID: m.ID, Items: []om.OrderItem{{DrinkID: d.ID, Quantity: 1}}})
}

func TestActiveContractsDoNotExposeDeletion(t *testing.T) {
	t.Parallel()
	for _, model := range []any{dm.Drink{}, im.Ingredient{}, mm.Menu{}, om.Order{}} {
		_, ok := reflect.TypeOf(model).FieldByName("DeletedAt")
		testutil.ErrorIf(t, ok, "active contract %T exposes deletion", model)
	}
}
