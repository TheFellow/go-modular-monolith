package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (d *FileStockDAO) Set(ctx context.Context, stock Stock) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !d.loaded {
		return errors.Internalf("dao not loaded")
	}
	if stock.IngredientID == "" {
		return errors.Invalidf("ingredient id is required")
	}

	for i, existing := range d.stock {
		if existing.IngredientID == stock.IngredientID {
			d.stock[i] = stock
			if mctx, ok := ctx.(*middleware.Context); ok {
				mctx.Cache().Set(stock)
			}
			return nil
		}
	}

	d.stock = append(d.stock, stock)
	if mctx, ok := ctx.(*middleware.Context); ok {
		mctx.Cache().Set(stock)
	}
	return nil
}
