package ingredients_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestIngredients_CreateGetUpdateDelete(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	count, err := f.Ingredients.Count(ctx, ingredients.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, count, 0)

	created, err := f.Ingredients.Create(ctx, &models.Ingredient{
		Name: "Lime Juice", Category: models.CategoryJuice,
		Unit: measurement.UnitOz, Description: "Fresh pressed lime",
	})
	testutil.Ok(t, err)
	testutil.IsFalse(t, created.ID.IsZero())

	got, err := f.Ingredients.Get(ctx, created.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, created)

	updated, err := f.Ingredients.Update(ctx, &models.Ingredient{
		ID: created.ID, Name: "Fresh Lime Juice", Unit: measurement.UnitMl,
	})
	testutil.Ok(t, err)
	wantUpdated := *created
	wantUpdated.Name = "Fresh Lime Juice"
	wantUpdated.Unit = measurement.UnitMl
	testutil.Equals(t, updated, &wantUpdated)

	got, err = f.Ingredients.Get(ctx, created.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, updated)

	deleted, err := f.Ingredients.Delete(ctx, created.ID)
	testutil.Ok(t, err)
	testutil.IsTrue(t, deleted.DeletedAt.IsSome())
	_, err = f.Ingredients.Get(ctx, created.ID)
	testutil.ErrorIsNotFound(t, err)
	count, err = f.Ingredients.Count(ctx, ingredients.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, count, 0)
}
