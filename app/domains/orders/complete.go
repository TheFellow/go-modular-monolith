package orders

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Complete(ctx *middleware.Context, order *models.Order) (*models.Order, error) {
	if order == nil {
		return nil, errors.Invalidf("order is required")
	}
	return m.pipeline.LoadCommand(ctx, authz.ActionComplete,
		func(ctx *middleware.Context) (*models.Order, error) {
			loaded, err := m.queries.Get(ctx, order.ID)
			if err != nil {
				return nil, err
			}
			if order.Revision != 0 && order.Revision != loaded.Revision {
				return nil, errors.Conflictf("order changed; reload before complete")
			}
			return loaded, nil
		},
		m.commands.Complete,
	)
}
