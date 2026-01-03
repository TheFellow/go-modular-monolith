package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (q *Queries) Get(ctx context.Context, id string) (models.Ingredient, error) {
	return middleware.Cached(ctx, "ingredients:get:"+id, func() (models.Ingredient, error) {
		record, ok, err := q.dao.Get(ctx, id)
		if err != nil {
			return models.Ingredient{}, errors.Internalf("get ingredient %s: %w", id, err)
		}
		if !ok {
			return models.Ingredient{}, errors.NotFoundf("ingredient %s not found", id)
		}
		return record.ToDomain(), nil
	})
}
