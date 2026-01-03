package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (q *Queries) Get(ctx context.Context, id string) (models.Drink, error) {
	return middleware.Cached(ctx, "drinks:get:"+id, func() (models.Drink, error) {
		record, ok, err := q.dao.Get(ctx, id)
		if err != nil {
			return models.Drink{}, errors.Internalf("get drink %s: %w", id, err)
		}
		if !ok {
			return models.Drink{}, errors.NotFoundf("drink %s not found", id)
		}
		return record.ToDomain(), nil
	})
}
