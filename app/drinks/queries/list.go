package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	perrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (q *Queries) List(ctx context.Context) ([]models.Drink, error) {
	records, err := q.dao.List(ctx)
	if err != nil {
		return nil, perrors.Internalf("list drinks: %w", err)
	}

	drinks := make([]models.Drink, 0, len(records))
	for _, record := range records {
		drinks = append(drinks, record.ToDomain())
	}
	return drinks, nil
}
