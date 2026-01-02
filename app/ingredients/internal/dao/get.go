package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (d *FileIngredientDAO) Get(ctx context.Context, id string) (Ingredient, bool, error) {
	if err := ctx.Err(); err != nil {
		return Ingredient{}, false, err
	}

	if !d.loaded {
		return Ingredient{}, false, errors.Internalf("dao not loaded")
	}

	for _, ingredient := range d.ingredients {
		if ingredient.ID != id {
			continue
		}
		if ingredient.DeletedAt != nil {
			return Ingredient{}, false, nil
		}
		return ingredient, true, nil
	}

	return Ingredient{}, false, nil
}
