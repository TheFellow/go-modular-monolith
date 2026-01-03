package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (d *FileIngredientDAO) Get(ctx context.Context, id string) (Ingredient, bool, error) {
	if err := ctx.Err(); err != nil {
		return Ingredient{}, false, err
	}

	if !d.loaded {
		return Ingredient{}, false, errors.Internalf("dao not loaded")
	}

	uid := models.NewIngredientID(id)
	if cached, ok := middleware.CacheGet[Ingredient](ctx, uid); ok {
		return cached, true, nil
	}

	for _, ingredient := range d.ingredients {
		if ingredient.ID != id {
			continue
		}
		if ingredient.DeletedAt != nil {
			return Ingredient{}, false, nil
		}
		middleware.CacheSet(ctx, ingredient)
		return ingredient, true, nil
	}

	return Ingredient{}, false, nil
}
