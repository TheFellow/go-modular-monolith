package ingredients_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
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

		existing := b.WithIngredient("Permissions Ingredient", models.UnitOz)

		_, err := a.Ingredients.List(owner, ingredients.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Ingredients.Get(owner, entity.IngredientID("does-not-exist"))
		testutil.RequireNotDenied(t, err)

		_, err = a.Ingredients.Create(owner, &models.Ingredient{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Ingredients.Update(owner, &models.Ingredient{ID: existing.ID, Description: "Updated"})
		testutil.RequireNotDenied(t, err)

		_, err = a.Ingredients.Delete(owner, existing.ID)
		testutil.RequireNotDenied(t, err)
	})

	t.Run("anonymous", func(t *testing.T) {
		t.Parallel()
		fix := testutil.NewFixture(t)
		b := fix.Bootstrap()
		a := fix.App
		anon := fix.ActorContext("anonymous")

		existing := b.WithIngredient("Permissions Ingredient", models.UnitOz)

		_, err := a.Ingredients.List(anon, ingredients.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Ingredients.Get(anon, entity.IngredientID("does-not-exist"))
		testutil.RequireNotDenied(t, err)

		_, err = a.Ingredients.Create(anon, &models.Ingredient{})
		testutil.RequireDenied(t, err)

		_, err = a.Ingredients.Update(anon, &models.Ingredient{ID: existing.ID, Description: "Updated"})
		testutil.RequireDenied(t, err)

		_, err = a.Ingredients.Delete(anon, existing.ID)
		testutil.RequireDenied(t, err)
	})
}
