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
	"github.com/TheFellow/go-modular-monolith/app/money"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

type fakeIngredients struct{}

func (fakeIngredients) Get(_ context.Context, _ cedar.EntityUID) (ingredientsmodels.Ingredient, error) {
	return ingredientsmodels.Ingredient{Unit: ingredientsmodels.UnitOz}, nil
}

func TestAdjust_EmitsStockAdjusted(t *testing.T) {
	testutil.OpenStore(t)

	d := dao.New()
	err := store.DB.Write(context.Background(), func(tx *bstore.Tx) error {
		ctx := middleware.NewContext(context.Background(), middleware.WithTransaction(tx))
		return d.Upsert(ctx, models.Stock{
			IngredientID: "vodka",
			Quantity:     1.0,
			Unit:         ingredientsmodels.UnitOz,
			CostPerUnit:  optional.None[money.Price](),
			LastUpdated:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		})
	})
	testutil.Ok(t, err)

	cmds := commands.NewWithDependencies(d, fakeIngredients{})
	ingredientID := ingredientsmodels.NewIngredientID("vodka")

	var evts []any
	err = store.DB.Write(context.Background(), func(tx *bstore.Tx) error {
		ctx := middleware.NewContext(context.Background(), middleware.WithTransaction(tx))
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
			testutil.ErrorIf(t, got.IngredientID != ingredientID, "unexpected ingredient id: %v", got.IngredientID)
			testutil.ErrorIf(t, got.PreviousQty != 1.0, "unexpected previous qty: %v", got.PreviousQty)
			testutil.ErrorIf(t, got.NewQty != 0.0, "unexpected new qty: %v", got.NewQty)
			testutil.ErrorIf(t, got.Delta != -1.0, "unexpected delta: %v", got.Delta)
		}
	}
	testutil.ErrorIf(t, !sawAdjusted, "expected StockAdjusted event")
}
