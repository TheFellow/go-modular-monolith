package ingredients_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPermissions_Ingredients(t *testing.T) {
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
			fix := testutil.NewFixture(t)
			b := fix.Bootstrap()
			a := fix.App
			var ctx *middleware.Context
			if tc.name == "owner" {
				ctx = fix.OwnerContext()
			} else {
				ctx = fix.ActorContext(tc.name)
			}

			existing := b.WithIngredient("Permissions Ingredient", measurement.UnitOz)

			_, err := a.Ingredients.List(ctx, ingredients.ListRequest{})
			testutil.PermissionTestPass(t, err)

			_, err = a.Ingredients.Get(ctx, entity.NewIngredientID())
			testutil.PermissionTestPass(t, err)

			_, err = a.Ingredients.Create(ctx, &models.Ingredient{})
			if tc.canWrite {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			_, err = a.Ingredients.Update(ctx, &models.Ingredient{ID: existing.ID, Description: "Updated"})
			if tc.canWrite {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}

			_, err = a.Ingredients.Delete(ctx, existing.ID)
			if tc.canWrite {
				testutil.PermissionTestPass(t, err)
			} else {
				testutil.PermissionTestFail(t, err)
			}
		})
	}
}
