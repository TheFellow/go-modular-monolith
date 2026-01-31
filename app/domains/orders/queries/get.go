package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) Get(ctx store.Context, id entity.OrderID) (*models.Order, error) {
	o, err := q.dao.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := o.Status.Validate(); err != nil {
		return nil, err
	}
	return o, nil
}
