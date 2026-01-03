package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (d *FileStockDAO) Get(ctx context.Context, ingredientID string) (Stock, bool, error) {
	if err := ctx.Err(); err != nil {
		return Stock{}, false, err
	}
	if !d.loaded {
		return Stock{}, false, errors.Internalf("dao not loaded")
	}

	uid := cedar.NewEntityUID(models.StockEntityType, cedar.String(ingredientID))
	if mctx, ok := ctx.(*middleware.Context); ok {
		if v, ok := mctx.Cache().Get(uid); ok {
			if cached, ok := v.(Stock); ok {
				return cached, true, nil
			}
		}
	}

	for _, s := range d.stock {
		if s.IngredientID != ingredientID {
			continue
		}
		if mctx, ok := ctx.(*middleware.Context); ok {
			mctx.Cache().Set(s)
		}
		return s, true, nil
	}
	return Stock{}, false, nil
}
