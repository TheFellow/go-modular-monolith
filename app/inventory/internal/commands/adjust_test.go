package commands_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/inventory/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/uow"
	cedar "github.com/cedar-policy/cedar-go"
)

type fakeIngredients struct{}

func (fakeIngredients) Get(_ context.Context, _ cedar.EntityUID) (ingredientsmodels.Ingredient, error) {
	return ingredientsmodels.Ingredient{Unit: ingredientsmodels.UnitOz}, nil
}

func TestAdjust_EmitsStockAdjusted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.json")

	const seed = `[
  { "ingredient_id": "vodka", "quantity": 1.0, "unit": "oz", "last_updated": "2026-01-01T00:00:00Z" }
]`

	err := os.WriteFile(path, []byte(seed), 0o644)
	testutil.ErrorIf(t, err != nil, "write seed: %v", err)

	d := dao.NewFileStockDAO(path)
	err = d.Load(context.Background())
	testutil.ErrorIf(t, err != nil, "load: %v", err)

	ctx := middleware.NewContext(context.Background())
	tx, err := uow.NewManager().Begin(ctx)
	testutil.ErrorIf(t, err != nil, "begin tx: %v", err)
	ctx = middleware.NewContext(ctx, middleware.WithUnitOfWork(tx))

	cmds := commands.NewWithDependencies(d, fakeIngredients{})
	ingredientID := ingredientsmodels.NewIngredientID("vodka")

	_, err = cmds.Adjust(ctx, models.StockAdjustment{
		IngredientID: ingredientID,
		Delta:        -2.0,
		Reason:       models.ReasonUsed,
	})
	testutil.ErrorIf(t, err != nil, "execute: %v", err)

	var sawAdjusted bool
	for _, e := range ctx.Events() {
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
