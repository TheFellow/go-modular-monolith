package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) Get(ctx context.Context, id cedar.EntityUID) (models.Ingredient, error) {
	ingredient, ok, err := q.dao.Get(ctx, id)
	if err != nil {
		return models.Ingredient{}, errors.Internalf("get ingredient %s: %w", id.ID, err)
	}
	if !ok {
		return models.Ingredient{}, errors.NotFoundf("ingredient %s not found", id.ID)
	}
	return ingredient, nil
}
