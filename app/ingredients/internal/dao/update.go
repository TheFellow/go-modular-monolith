package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (d *FileIngredientDAO) Update(ctx context.Context, ingredient Ingredient) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !d.loaded {
		return errors.Internalf("dao not loaded")
	}
	if ingredient.ID == "" {
		return errors.Invalidf("ingredient id is required")
	}

	for i, existing := range d.ingredients {
		if existing.ID == ingredient.ID {
			if existing.DeletedAt != nil {
				return errors.NotFoundf("ingredient %s not found", ingredient.ID)
			}
			d.ingredients[i] = ingredient
			middleware.CacheSet(ctx, ingredient)
			return nil
		}
	}

	return errors.NotFoundf("ingredient %s not found", ingredient.ID)
}
