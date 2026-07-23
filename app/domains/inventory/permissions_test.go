package inventory_test

import (
	"testing"

	ingredientsM "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventoryM "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPermissions_Inventory(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		canWrite bool
	}{
		{name: "owner", canWrite: true},
		{name: "manager", canWrite: true},
		{name: "sommelier", canWrite: false},
		{name: "bartender", canWrite: false},
		{name: "anonymous", canWrite: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := testutil.NewFixture(t)
			a := f.App
			owner := f.OwnerContext()
			var ctx *middleware.Context
			if tc.name == "owner" {
				ctx = owner
			} else {
				ctx = f.ActorContext(tc.name)
			}
			ingredient := testutil.CreateIngredient(t, f, ingredientsM.Ingredient{
				Name: "Inventory Permissions Ingredient", Category: ingredientsM.CategoryOther, Unit: measurement.UnitOz,
			})
			testutil.SetInventory(t, f, inventoryM.Update{
				IngredientID: ingredient.ID,
				Amount:       measurement.MustAmount(10, ingredient.Unit),
				CostPerUnit:  money.NewPriceFromCents(100, currency.USD),
			})

			_, err := a.Inventory.List(ctx, inventory.ListRequest{})
			testutil.Ok(t, err)

			_, err = a.Inventory.Get(ctx, ingredient.ID)
			testutil.Ok(t, err)

			_, err = a.Inventory.Adjust(ctx, &inventoryM.Patch{
				IngredientID: ingredient.ID,
				Delta:        optional.Some(measurement.MustAmount(1, measurement.UnitOz)),
				Reason:       inventoryM.ReasonCorrected,
			})
			if tc.canWrite {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}
			adjusted, err := a.Inventory.Get(owner, ingredient.ID)
			testutil.Ok(t, err)
			wantAdjusted := 10.0
			if tc.canWrite {
				wantAdjusted = 11
			}
			testutil.Equals(t, adjusted.Amount, measurement.MustAmount(wantAdjusted, ingredient.Unit))

			_, err = a.Inventory.Set(ctx, &inventoryM.Update{
				IngredientID: ingredient.ID,
				Amount:       measurement.MustAmount(20, measurement.UnitOz),
				CostPerUnit:  money.NewPriceFromCents(100, currency.USD),
			})
			if tc.canWrite {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}
			set, err := a.Inventory.Get(owner, ingredient.ID)
			testutil.Ok(t, err)
			wantSet := 10.0
			if tc.canWrite {
				wantSet = 20
			}
			testutil.Equals(t, set.Amount, measurement.MustAmount(wantSet, ingredient.Unit))
		})
	}
}
