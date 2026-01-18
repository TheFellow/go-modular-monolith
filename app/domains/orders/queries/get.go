package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) Get(ctx dao.Context, id cedar.EntityUID) (*models.Order, error) {
	o, err := q.dao.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := o.Status.Validate(); err != nil {
		return nil, err
	}
	return o, nil
}
