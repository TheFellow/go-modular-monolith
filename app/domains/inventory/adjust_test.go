package inventory_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestInventory_Adjust(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	ingredient := b.WithIngredient("Vodka", measurement.UnitOz)

	_, err := f.Inventory.Set(ctx, &models.Update{
		IngredientID: ingredient.ID,
		Amount:       measurement.MustAmount(1.0, ingredient.Unit),
		CostPerUnit:  money.NewPriceFromCents(100, currency.USD),
	})
	testutil.Ok(t, err)

	stock, err := f.Inventory.Adjust(ctx, &models.Patch{
		IngredientID: ingredient.ID,
		Reason:       models.ReasonUsed,
		Delta:        optional.Some(measurement.MustAmount(-2.0, ingredient.Unit)),
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, stock.Amount.Value() != 0.0, "expected quantity 0.0, got %v", stock.Amount.Value())
}
