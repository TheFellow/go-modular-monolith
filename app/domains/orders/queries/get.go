package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) Get(ctx context.Context, id cedar.EntityUID) (*models.Order, error) {
	o, err := q.dao.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := o.Status.Validate(); err != nil {
		return nil, err
	}
	return o, nil
}
