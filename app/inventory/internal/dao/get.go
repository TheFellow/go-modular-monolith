package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (d *FileStockDAO) Get(ctx context.Context, ingredientID string) (Stock, bool, error) {
	if err := ctx.Err(); err != nil {
		return Stock{}, false, err
	}
	if !d.loaded {
		return Stock{}, false, errors.Internalf("dao not loaded")
	}

	for _, s := range d.stock {
		if s.IngredientID != ingredientID {
			continue
		}
		return s, true, nil
	}
	return Stock{}, false, nil
}
