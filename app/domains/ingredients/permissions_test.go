package ingredients_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
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
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fix := testutil.NewFixture(t)
			a := fix.App
			owner := fix.OwnerContext()
			var ctx *middleware.Context
			if tc.name == "owner" {
				ctx = owner
			} else {
				ctx = fix.ActorContext(tc.name)
			}

			existing := testutil.CreateIngredient(t, fix, models.Ingredient{
				Name: "Permissions Ingredient", Category: models.CategoryOther, Unit: measurement.UnitOz,
			})

			_, err := a.Ingredients.List(ctx, ingredients.ListRequest{})
			testutil.Ok(t, err)

			_, err = a.Ingredients.Get(ctx, existing.ID)
			testutil.Ok(t, err)

			_, err = a.Ingredients.Create(ctx, &models.Ingredient{
				Name: "Created Ingredient", Category: models.CategoryOther, Unit: measurement.UnitOz,
			})
			if tc.canWrite {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}
			count, err := a.Ingredients.Count(owner, ingredients.ListRequest{})
			testutil.Ok(t, err)
			wantCount := 1
			if tc.canWrite {
				wantCount = 2
			}
			testutil.Equals(t, count, wantCount)

			update := *existing
			update.Description = "Updated"
			_, err = a.Ingredients.Update(ctx, &update)
			if tc.canWrite {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}
			got, err := a.Ingredients.Get(owner, existing.ID)
			testutil.Ok(t, err)
			wantDescription := ""
			if tc.canWrite {
				wantDescription = "Updated"
			}
			testutil.Equals(t, got.Description, wantDescription)

			_, err = a.Ingredients.Delete(ctx, existing.ID)
			if tc.canWrite {
				testutil.Ok(t, err)
			} else {
				testutil.ErrorIsPermission(t, err)
			}
			_, err = a.Ingredients.Get(owner, existing.ID)
			if tc.canWrite {
				testutil.ErrorIsNotFound(t, err)
			} else {
				testutil.Ok(t, err)
			}
		})
	}
}
