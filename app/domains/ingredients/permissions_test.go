package ingredients_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPermissions_Ingredients(t *testing.T) {
	fix := testutil.NewFixture(t)
	a := fix.App

	owner := fix.Ctx
	anon := fix.AsActor("anonymous")

	t.Run("owner", func(t *testing.T) {
		_, err := a.Ingredients.List(owner, ingredients.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Ingredients.Get(owner, ingredients.GetRequest{ID: entity.IngredientID("does-not-exist")})
		testutil.RequireNotDenied(t, err)

		_, err = a.Ingredients.Create(owner, models.Ingredient{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Ingredients.Update(owner, models.Ingredient{ID: entity.IngredientID("does-not-exist")})
		testutil.RequireNotDenied(t, err)
	})

	t.Run("anonymous", func(t *testing.T) {
		_, err := a.Ingredients.List(anon, ingredients.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Ingredients.Get(anon, ingredients.GetRequest{ID: entity.IngredientID("does-not-exist")})
		testutil.RequireNotDenied(t, err)

		_, err = a.Ingredients.Create(anon, models.Ingredient{})
		testutil.RequireDenied(t, err)

		_, err = a.Ingredients.Update(anon, models.Ingredient{ID: entity.IngredientID("does-not-exist")})
		testutil.RequireDenied(t, err)
	})
}
