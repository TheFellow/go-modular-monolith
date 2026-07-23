package testutil

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func TestFixtureContextsShareWrappingTransaction(t *testing.T) {
	t.Parallel()
	f := NewFixture(t)

	want, ok := f.OwnerContext().Transaction()
	IsTrue(t, ok)
	actorTx, ok := f.ActorContext("manager").Transaction()
	IsTrue(t, ok)
	sessionTx, ok := f.App.Context().Transaction()
	IsTrue(t, ok)
	IsTrue(t, actorTx == want)
	IsTrue(t, sessionTx == want)
}

func TestFixtureRollbackDiscardsApplicationWrites(t *testing.T) {
	t.Parallel()
	f := NewFixture(t)

	_, err := f.Ingredients.Create(f.OwnerContext(), &models.Ingredient{
		Name: "Transient Gin", Category: models.CategorySpirit, Unit: measurement.UnitOz,
	})
	Ok(t, err)
	Ok(t, f.rollback())

	ctx := middleware.NewContext(f.ctx)
	page, err := f.Ingredients.List(ctx, ingredients.ListRequest{})
	Ok(t, err)
	Equals(t, len(page.Items), 0)
}
