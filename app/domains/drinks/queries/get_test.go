package queries_test

import (
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/mjl-/bstore"
)

func TestGet_Found(t *testing.T) {
	fix := testutil.NewFixture(t)

	d := dao.New()
	err := fix.Store.Write(context.Background(), func(tx *bstore.Tx) error {
		ctx := middleware.NewContext(fix.Ctx, middleware.WithTransaction(tx))
		return d.Insert(ctx, models.Drink{
			ID:   models.NewDrinkID("margarita"),
			Name: "Margarita",
		})
	})
	testutil.Ok(t, err)

	q := queries.NewWithDAO(d)

	got, err := q.Get(fix.Ctx, models.NewDrinkID("margarita"))
	testutil.ErrorIf(t, err != nil, "get: %v", err)

	testutil.Equals(t, got, models.Drink{ID: models.NewDrinkID("margarita"), Name: "Margarita"})
}

func TestGet_NotFound(t *testing.T) {
	fix := testutil.NewFixture(t)
	q := queries.NewWithDAO(dao.New())

	_, err := q.Get(fix.Ctx, models.NewDrinkID("missing"))
	testutil.ErrorIf(t, !errors.IsNotFound(err), "expected NotFound error, got %v", err)
}
