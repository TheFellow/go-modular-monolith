package queries

import (
	"iter"

	ordersdao "github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) List(ctx store.Context, filter ordersdao.ListFilter) iter.Seq2[*models.Order, error] {
	return q.dao.List(ctx, filter)
}
