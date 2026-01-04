package dao

import (
	"context"
)

func (d *FileIngredientDAO) Get(ctx context.Context, id string) (Ingredient, bool, error) {
	if err := ctx.Err(); err != nil {
		return Ingredient{}, false, err
	}

	if err := d.ensureLoaded(ctx); err != nil {
		return Ingredient{}, false, err
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
