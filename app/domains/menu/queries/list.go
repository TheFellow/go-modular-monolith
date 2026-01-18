package queries

import (
	"sort"

	menudao "github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) List(ctx store.Context, filter menudao.ListFilter) ([]*models.Menu, error) {
	out, err := q.dao.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
