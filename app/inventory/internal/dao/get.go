package dao

import (
	"context"
)

func (d *FileStockDAO) Get(ctx context.Context, ingredientID string) (Stock, bool, error) {
	if err := ctx.Err(); err != nil {
		return Stock{}, false, err
	}
	if err := d.ensureLoaded(ctx); err != nil {
		return Stock{}, false, err
	}

	for _, s := range d.stock {
		if s.IngredientID != ingredientID {
			continue
		}
		return s, true, nil
	}
	return Stock{}, false, nil
}
