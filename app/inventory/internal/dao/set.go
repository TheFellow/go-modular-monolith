package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (d *FileStockDAO) Set(ctx context.Context, stock Stock) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := d.ensureLoaded(ctx); err != nil {
		return err
	}
	if stock.IngredientID == "" {
		return errors.Invalidf("ingredient id is required")
	}

	for i, existing := range d.stock {
		if existing.IngredientID == stock.IngredientID {
			d.stock[i] = stock
			return nil
		}
	}

	d.stock = append(d.stock, stock)
	return nil
}
