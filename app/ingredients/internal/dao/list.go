package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (d *FileIngredientDAO) List(ctx context.Context) ([]Ingredient, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if !d.loaded {
		return nil, errors.Internalf("dao not loaded")
	}

	out := make([]Ingredient, 0, len(d.ingredients))
	for _, ingredient := range d.ingredients {
		if ingredient.DeletedAt != nil {
			continue
		}
		out = append(out, ingredient)
	}
	return out, nil
}
