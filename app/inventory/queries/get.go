package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) Get(ctx context.Context, ingredientID cedar.EntityUID) (models.Stock, error) {
	id := string(ingredientID.ID)
	record, ok, err := q.dao.Get(ctx, id)
	if err != nil {
		return models.Stock{}, errors.Internalf("get stock %s: %w", id, err)
	}
	if !ok {
		return models.Stock{}, errors.NotFoundf("stock %s not found", id)
	}
	return record.ToDomain(), nil
}
