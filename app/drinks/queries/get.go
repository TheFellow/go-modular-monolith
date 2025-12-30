package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	perrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (q *Queries) Get(ctx context.Context, id string) (models.Drink, error) {
	record, ok, err := q.dao.Get(ctx, id)
	if err != nil {
		return models.Drink{}, perrors.Internalf("get drink %s: %w", id, err)
	}
	if !ok {
		return models.Drink{}, perrors.NotFoundf("drink %s not found", id)
	}
	return record.ToDomain(), nil
}
