package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (q *Queries) List(ctx context.Context) ([]models.Ingredient, error) {
	records, err := q.dao.List(ctx)
	if err != nil {
		return nil, errors.Internalf("list ingredients: %w", err)
	}

	ingredients := make([]models.Ingredient, 0, len(records))
	for _, record := range records {
		ingredients = append(ingredients, record.ToDomain())
	}
	return ingredients, nil
}
