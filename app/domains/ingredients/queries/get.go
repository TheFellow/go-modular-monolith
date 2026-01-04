package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) Get(ctx context.Context, id cedar.EntityUID) (models.Ingredient, error) {
	record, ok, err := q.dao.Get(ctx, string(id.ID))
	if err != nil {
		return models.Ingredient{}, errors.Internalf("get ingredient %s: %w", id.ID, err)
	}
	if !ok {
		return models.Ingredient{}, errors.NotFoundf("ingredient %s not found", id.ID)
	}
	return record.ToDomain(), nil
}
