package inventory_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventoryM "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPermissions_Inventory(t *testing.T) {
	t.Parallel()

	t.Run("owner", func(t *testing.T) {
		t.Parallel()
		f := testutil.NewFixture(t)
		a := f.App
		owner := f.OwnerContext()
		missingID := entity.NewIngredientID()

		_, err := a.Inventory.List(owner, inventory.ListRequest{})
		testutil.PermissionTestPass(t, err)

		_, err = a.Inventory.Get(owner, missingID)
		testutil.PermissionTestPass(t, err)

		_, err = a.Inventory.Adjust(owner, &inventoryM.Patch{
			IngredientID: missingID,
			Delta:        optional.Some(measurement.MustAmount(1, measurement.UnitOz)),
			Reason:       inventoryM.ReasonCorrected,
		})
		testutil.PermissionTestPass(t, err)

		_, err = a.Inventory.Set(owner, &inventoryM.Update{
			IngredientID: missingID,
			Amount:       measurement.MustAmount(1, measurement.UnitOz),
		})
		testutil.PermissionTestPass(t, err)
	})

	t.Run("anonymous", func(t *testing.T) {
		t.Parallel()
		f := testutil.NewFixture(t)
		a := f.App
		anon := f.ActorContext("anonymous")
		missingID := entity.NewIngredientID()

		_, err := a.Inventory.List(anon, inventory.ListRequest{})
		testutil.PermissionTestPass(t, err)

		_, err = a.Inventory.Get(anon, missingID)
		testutil.PermissionTestPass(t, err)

		_, err = a.Inventory.Adjust(anon, &inventoryM.Patch{
			IngredientID: missingID,
			Delta:        optional.Some(measurement.MustAmount(1, measurement.UnitOz)),
			Reason:       inventoryM.ReasonCorrected,
		})
		testutil.PermissionTestFail(t, err)

		_, err = a.Inventory.Set(anon, &inventoryM.Update{
			IngredientID: missingID,
			Amount:       measurement.MustAmount(1, measurement.UnitOz),
		})
		testutil.PermissionTestFail(t, err)
	})
}
