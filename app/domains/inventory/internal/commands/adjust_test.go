package commands_test

import (
	"context"
	"testing"
	"time"

	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

type fakeIngredients struct{}

func (fakeIngredients) Get(_ context.Context, _ cedar.EntityUID) (ingredientsmodels.Ingredient, error) {
	return ingredientsmodels.Ingredient{Unit: ingredientsmodels.UnitOz}, nil
}

func TestAdjust_EmitsStockAdjusted(t *testing.T) {
	fix := testutil.NewFixture(t)

	d := dao.New()
	err := fix.Store.Write(context.Background(), func(tx *bstore.Tx) error {
		ctx := middleware.NewContext(fix.Ctx, middleware.WithTransaction(tx))
		return d.Upsert(ctx, models.Stock{
			IngredientID: entity.IngredientID("vodka"),
			Quantity:     1.0,
			Unit:         ingredientsmodels.UnitOz,
			CostPerUnit:  optional.None[money.Price](),
			LastUpdated:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		})
	})
	testutil.Ok(t, err)

	cmds := commands.NewWithDependencies(d, fakeIngredients{})
	ingredientID := entity.IngredientID("vodka")

	var evts []any
	err = fix.Store.Write(context.Background(), func(tx *bstore.Tx) error {
		ctx := middleware.NewContext(fix.Ctx, middleware.WithTransaction(tx))
		_, err := cmds.Adjust(ctx, models.StockPatch{
			IngredientID: ingredientID,
			Delta:        optional.Some(-2.0),
			Reason:       models.ReasonUsed,
		})
		evts = ctx.Events()
		return err
	})
	testutil.Ok(t, err)

	var sawAdjusted bool
	for _, e := range evts {
		switch got := e.(type) {
		case events.StockAdjusted:
			sawAdjusted = true
			testutil.ErrorIf(t, got.Current.IngredientID != ingredientID, "unexpected ingredient id: %v", got.Current.IngredientID)
			testutil.ErrorIf(t, got.Previous.Quantity != 1.0, "unexpected previous qty: %v", got.Previous.Quantity)
			testutil.ErrorIf(t, got.Current.Quantity != 0.0, "unexpected new qty: %v", got.Current.Quantity)
			testutil.ErrorIf(t, got.Current.Quantity-got.Previous.Quantity != -1.0, "unexpected delta: %v", got.Current.Quantity-got.Previous.Quantity)
			testutil.ErrorIf(t, got.Reason != "used", "unexpected reason: %v", got.Reason)
		}
	}
	testutil.ErrorIf(t, !sawAdjusted, "expected StockAdjusted event")
}
