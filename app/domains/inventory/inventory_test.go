package inventory_test

import (
	"testing"

	ingredientsM "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestInventory_SetAndAdjust(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	ingredient := b.WithIngredient("Vodka", ingredientsM.UnitOz)

	stock, err := f.Inventory.Set(ctx, &models.Update{
		IngredientID: ingredient.ID,
		Quantity:     1.0,
		CostPerUnit:  money.NewPriceFromCents(100, "USD"),
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, stock.Quantity != 1.0, "expected quantity 1.0, got %v", stock.Quantity)

	stock, err = f.Inventory.Adjust(ctx, &models.Patch{
		IngredientID: ingredient.ID,
		Reason:       models.ReasonUsed,
		Delta:        optional.Some(-2.0),
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, stock.Quantity != 0.0, "expected quantity 0.0, got %v", stock.Quantity)
}
