package queries

import (
	"context"
	"sort"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
)

func (q *Queries) List(ctx context.Context, filter dao.ListFilter) ([]*models.Menu, error) {
	out, err := q.dao.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
