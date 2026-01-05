package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (q *Queries) List(ctx context.Context, filter dao.ListFilter) ([]models.Drink, error) {
	drinks, err := q.dao.List(ctx, filter)
	if err != nil {
		return nil, errors.Internalf("list drinks: %w", err)
	}

	return drinks, nil
}
