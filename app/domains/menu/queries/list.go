package queries

import (
	"context"
	"sort"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (q *Queries) List(ctx context.Context) ([]models.Menu, error) {
	records, err := q.dao.List(ctx)
	if err != nil {
		return nil, errors.Internalf("list menus: %w", err)
	}

	out := make([]models.Menu, 0, len(records))
	for _, m := range records {
		out = append(out, m.ToDomain())
	}

	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
