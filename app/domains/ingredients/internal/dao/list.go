package dao

import (
	"context"
)

func (d *FileIngredientDAO) List(ctx context.Context) ([]Ingredient, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := d.ensureLoaded(ctx); err != nil {
		return nil, err
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
