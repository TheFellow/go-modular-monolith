package inventory_test

import (
	"testing"

	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestInventory_SetGetAdjustAndRemove(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "London Dry Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz,
	})

	count, err := f.Inventory.Count(ctx, inventory.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, count, 0)

	cost := money.NewPriceFromCents(250, currency.USD)
	set, err := f.Inventory.Set(ctx, &models.Update{
		IngredientID: ingredient.ID,
		Amount:       measurement.MustAmount(10, ingredient.Unit),
		CostPerUnit:  cost,
	})
	testutil.Ok(t, err)
	testutil.IsFalse(t, set.ID.IsZero())
	wantSet := &models.Inventory{
		ID: set.ID, IngredientID: ingredient.ID,
		Amount: measurement.MustAmount(10, ingredient.Unit), CostPerUnit: optional.Some(cost),
		LastUpdated: set.LastUpdated,
	}
	testutil.Equals(t, set, wantSet)

	got, err := f.Inventory.Get(ctx, ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, set)

	adjusted, err := f.Inventory.Adjust(ctx, &models.Patch{
		IngredientID: ingredient.ID,
		Reason:       models.ReasonReceived,
		Delta:        optional.Some(measurement.MustAmount(2.5, ingredient.Unit)),
	})
	testutil.Ok(t, err)
	wantAdjusted := *set
	wantAdjusted.Amount = measurement.MustAmount(12.5, ingredient.Unit)
	wantAdjusted.LastUpdated = adjusted.LastUpdated
	testutil.Equals(t, adjusted, &wantAdjusted)

	got, err = f.Inventory.Get(ctx, ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, adjusted)

	_, err = f.Ingredients.Delete(ctx, ingredient.ID)
	testutil.Ok(t, err)
	_, err = f.Inventory.Get(ctx, ingredient.ID)
	testutil.ErrorIsNotFound(t, err)
	count, err = f.Inventory.Count(ctx, inventory.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, count, 0)
}
