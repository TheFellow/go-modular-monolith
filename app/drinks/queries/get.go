package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) Get(ctx context.Context, id cedar.EntityUID) (models.Drink, error) {
	return middleware.CachedByUID(ctx, id, func() (models.Drink, error) {
		record, ok, err := q.dao.Get(ctx, string(id.ID))
		if err != nil {
			return models.Drink{}, errors.Internalf("get drink %s: %w", id.ID, err)
		}
		if !ok {
			return models.Drink{}, errors.NotFoundf("drink %s not found", id.ID)
		}
		return record.ToDomain(), nil
	})
}
