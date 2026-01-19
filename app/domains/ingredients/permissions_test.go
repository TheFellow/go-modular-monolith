package ingredients_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPermissions_Ingredients(t *testing.T) {
	t.Parallel()

	t.Run("owner", func(t *testing.T) {
		t.Parallel()
		fix := testutil.NewFixture(t)
		b := fix.Bootstrap()
		a := fix.App
		owner := fix.OwnerContext()

		existing := b.WithIngredient("Permissions Ingredient", measurement.UnitOz)

		_, err := a.Ingredients.List(owner, ingredients.ListRequest{})
		testutil.PermissionTestPass(t, err)

		_, err = a.Ingredients.Get(owner, entity.NewIngredientID())
		testutil.PermissionTestPass(t, err)

		_, err = a.Ingredients.Create(owner, &models.Ingredient{})
		testutil.PermissionTestPass(t, err)

		_, err = a.Ingredients.Update(owner, &models.Ingredient{ID: existing.ID, Description: "Updated"})
		testutil.PermissionTestPass(t, err)

		_, err = a.Ingredients.Delete(owner, existing.ID)
		testutil.PermissionTestPass(t, err)
	})

	t.Run("anonymous", func(t *testing.T) {
		t.Parallel()
		fix := testutil.NewFixture(t)
		b := fix.Bootstrap()
		a := fix.App
		anon := fix.ActorContext("anonymous")

		existing := b.WithIngredient("Permissions Ingredient", measurement.UnitOz)

		_, err := a.Ingredients.List(anon, ingredients.ListRequest{})
		testutil.PermissionTestPass(t, err)

		_, err = a.Ingredients.Get(anon, entity.NewIngredientID())
		testutil.PermissionTestPass(t, err)

		_, err = a.Ingredients.Create(anon, &models.Ingredient{})
		testutil.PermissionTestFail(t, err)

		_, err = a.Ingredients.Update(anon, &models.Ingredient{ID: existing.ID, Description: "Updated"})
		testutil.PermissionTestFail(t, err)

		_, err = a.Ingredients.Delete(anon, existing.ID)
		testutil.PermissionTestFail(t, err)
	})
}
