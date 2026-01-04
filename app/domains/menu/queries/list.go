package queries

import (
	"context"
	"sort"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (q *Queries) List(ctx context.Context) ([]models.Menu, error) {
	out, err := q.dao.List(ctx)
	if err != nil {
		return nil, errors.Internalf("list menus: %w", err)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
