package queries

import (
	"sort"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (q *Queries) List(ctx *middleware.Context) ([]models.Order, error) {
	records, err := q.dao.List(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]models.Order, 0, len(records))
	for _, r := range records {
		out = append(out, r.ToDomain())
	}

	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}
