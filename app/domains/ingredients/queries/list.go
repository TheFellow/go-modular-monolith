package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (q *Queries) List(ctx context.Context) ([]models.Ingredient, error) {
	ingredients, err := q.dao.List(ctx)
	if err != nil {
		return nil, errors.Internalf("list ingredients: %w", err)
	}

	return ingredients, nil
}
