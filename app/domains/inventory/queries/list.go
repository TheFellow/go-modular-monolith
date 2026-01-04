package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (q *Queries) List(ctx context.Context) ([]models.Stock, error) {
	records, err := q.dao.List(ctx)
	if err != nil {
		return nil, errors.Internalf("list stock: %w", err)
	}

	out := make([]models.Stock, 0, len(records))
	for _, record := range records {
		out = append(out, record.ToDomain())
	}
	return out, nil
}
