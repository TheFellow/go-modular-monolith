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
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"math"
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
func TestCanonicalStockSurvivesDisplayUnitChange(t *testing.T) {
	t.Parallel()
	f, i, d, m := workflowFixture(t)
	ctx := f.OwnerContext()
	workflowOrder(t, f, d, m)
	i.Unit = measurement.UnitMl
	_, err := f.Ingredients.Update(ctx, i)
	testutil.Ok(t, err)
	stock, err := f.Inventory.Adjust(ctx, &iv.Patch{IngredientID: i.ID, Reason: iv.ReasonCorrected, Delta: optional.Some[measurement.Amount](measurement.MustAmount(0, i.Unit))})
	testutil.Ok(t, err)
	reserved, err := stock.ReservedAmount().Convert(measurement.UnitOz)
	testutil.Ok(t, err)
	testutil.Equals(t, reserved.Value(), 2.0)
	testutil.Equals(t, stock.CostUnit, measurement.UnitOz)
	order := workflowOrder(t, f, d, m)
	_, err = f.Orders.Complete(ctx, order)
	testutil.Ok(t, err)
	stock, err = f.Inventory.Get(ctx, i.ID)
	testutil.Ok(t, err)
	remaining, err := stock.Amount.Convert(measurement.UnitOz)
	testutil.Ok(t, err)
	testutil.IsTrue(t, math.Abs(remaining.Value()-8) < 1e-9)
}
func TestDiscontinuationHonorsAcceptedStockAndDisposalKeepsEvidence(t *testing.T) {
	t.Parallel()
	f, i, d, m := workflowFixture(t)
	ctx := f.OwnerContext()
	order := workflowOrder(t, f, d, m)
	_, err := f.Ingredients.Retire(ctx, i.ID, im.Retirement{Reason: "discontinued"})
	testutil.Ok(t, err)
	_, err = f.Orders.Complete(ctx, order)
	testutil.Ok(t, err)
	stock, err := f.Inventory.Get(ctx, i.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Status, iv.StatusDiscontinued)
	_, err = f.Inventory.Dispose(ctx, iv.Disposal{IngredientID: i.ID, Revision: stock.Revision, Amount: stock.Amount, Reason: "discard remaining stock"})
	testutil.Ok(t, err)
	history, err := f.Inventory.History(ctx, i.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, history[len(history)-1].After, 0.0)
}
func TestReferencedDrinkDeletionFailsAtomically(t *testing.T) {
	t.Parallel()
	f, _, d, m := workflowFixture(t)
	_, err := f.Drinks.Delete(f.OwnerContext(), d.ID)
	testutil.ErrorIsFailedPrecondition(t, err)
	testutil.ErrorContains(t, err, m.Name)
	_, err = f.Drinks.Get(f.OwnerContext(), d.ID)
	testutil.Ok(t, err)
}
func TestOptionalIngredientIsReservedAndConsumed(t *testing.T) {
	t.Parallel()
	f, _, d, m := workflowFixture(t)
	ctx := f.OwnerContext()
	garnish := testutil.CreateIngredient(t, f, im.Ingredient{Name: "Garnish", Category: im.CategoryGarnish, Unit: measurement.UnitPiece})
	testutil.SetInventory(t, f, workflowStock(garnish, 3))
	d.Recipe.Ingredients = append(d.Recipe.Ingredients, dm.RecipeIngredient{IngredientID: garnish.ID, Amount: measurement.MustAmount(1, garnish.Unit), Optional: true})
	d, err := f.Drinks.Update(ctx, d)
	testutil.Ok(t, err)
	order := workflowOrder(t, f, d, m)
	stock, err := f.Inventory.Get(ctx, garnish.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.ReservedAmount().Value(), 1.0)
	_, err = f.Orders.Complete(ctx, order)
	testutil.Ok(t, err)
	stock, err = f.Inventory.Get(ctx, garnish.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Amount.Value(), 2.0)
}
func TestStockAndTagEditorsRejectStaleState(t *testing.T) {
	t.Parallel()
	f, i, _, _ := workflowFixture(t)
	ctx := f.OwnerContext()
	stock, err := f.Inventory.Get(ctx, i.ID)
	testutil.Ok(t, err)
	testutil.SetInventory(t, f, workflowStock(i, 8))
	update := workflowStock(i, 20)
	update.Revision = stock.Revision
	_, err = f.Inventory.Set(ctx, &update)
	testutil.ErrorIsConflict(t, err)
	before := tag.Tags{}
	_, err = f.App.Tags.Upsert(ctx, i.EntityUID(), tag.Tag{Key: "new"})
	testutil.Ok(t, err)
	_, err = f.App.Tags.Replace(ctx, i.EntityUID(), tag.Tags{{Key: "stale"}}, before)
	testutil.ErrorIsConflict(t, err)
}
