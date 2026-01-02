package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (d *FileIngredientDAO) Add(ctx context.Context, ingredient Ingredient) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !d.loaded {
		return errors.Internalf("dao not loaded")
	}
	if ingredient.ID == "" {
		return errors.Invalidf("ingredient id is required")
	}
	if ingredient.Name == "" {
		return errors.Invalidf("ingredient name is required")
	}

	for _, existing := range d.ingredients {
		if existing.ID == ingredient.ID {
			return errors.Invalidf("ingredient already exists: %s", ingredient.ID)
		}
	}

	d.ingredients = append(d.ingredients, ingredient)
	return nil
}
