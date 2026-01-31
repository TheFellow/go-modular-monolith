package inventory_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventoryM "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := testutil.NewFixture(t)
			a := f.App
			var ctx *middleware.Context
			if tc.name == "owner" {
				ctx = f.OwnerContext()
			} else {
				ctx = f.ActorContext(tc.name)
			}
			missingID := entity.NewIngredientID()

			_, err := a.Inventory.List(ctx, inventory.ListRequest{})
			testutil.PermissionTestPass(t, err)

			_, err = a.Inventory.Get(ctx, missingID)
			testutil.PermissionTestPass(t, err)

			_, err = a.Inventory.Adjust(ctx, &inventoryM.Patch{
				IngredientID: missingID,
				Delta:        optional.Some(measurement.MustAmount(1, measurement.UnitOz)),
				Reason:       inventoryM.ReasonCorrected,
			})
			if tc.canWrite {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			_, err = a.Inventory.Set(ctx, &inventoryM.Update{
				IngredientID: missingID,
				Amount:       measurement.MustAmount(1, measurement.UnitOz),
			})
			if tc.canWrite {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}
		})
	}
}
