package queries

import (
	"sort"

	ordersdao "github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) List(ctx store.Context, filter ordersdao.ListFilter) ([]*models.Order, error) {
	out, err := q.dao.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}
