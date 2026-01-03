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
)

func TestAdjust_EmitsDepletedAndRestocked(t *testing.T) {
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

	uc := commands.NewAdjust(d)
	ingredientID := ingredientsmodels.NewIngredientID("vodka")

	// Deplete
	_, err = uc.Execute(ctx, commands.AdjustRequest{
		IngredientID: ingredientID,
		Delta:        -2.0,
		Unit:         ingredientsmodels.UnitOz,
		Reason:       models.ReasonUsed,
	})
	testutil.ErrorIf(t, err != nil, "execute: %v", err)

	// Restock
	_, err = uc.Execute(ctx, commands.AdjustRequest{
		IngredientID: ingredientID,
		Delta:        5.0,
		Unit:         ingredientsmodels.UnitOz,
		Reason:       models.ReasonReceived,
	})
	testutil.ErrorIf(t, err != nil, "execute: %v", err)

	var sawDepleted bool
	var sawRestocked bool
	for _, e := range ctx.Events() {
		switch e.(type) {
		case events.IngredientDepleted:
			sawDepleted = true
		case events.IngredientRestocked:
			sawRestocked = true
		}
	}
	testutil.ErrorIf(t, !sawDepleted, "expected IngredientDepleted event")
	testutil.ErrorIf(t, !sawRestocked, "expected IngredientRestocked event")
}
