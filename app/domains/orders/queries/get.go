package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) Get(ctx *middleware.Context, id cedar.EntityUID) (models.Order, error) {
	if string(id.ID) == "" {
		return models.Order{}, errors.Invalidf("id is required")
	}

	record, found, err := q.dao.Get(ctx, string(id.ID))
	if err != nil {
		return models.Order{}, err
	}
	if !found {
		return models.Order{}, errors.NotFoundf("order %q not found", id.ID)
	}

	o := record.ToDomain()
	if err := o.Status.Validate(); err != nil {
		return models.Order{}, err
	}
	return o, nil
}
