package orders

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Complete(ctx *middleware.Context, order models.Order) (*models.Order, error) {
	return middleware.RunCommand(ctx, authz.ActionComplete,
		func(ctx *middleware.Context) (models.Order, error) {
			current, err := m.queries.Get(ctx, order.ID)
			if err != nil {
				return models.Order{}, err
			}
			return *current, nil
		},
		func(ctx *middleware.Context, current models.Order) (*models.Order, error) {
			return m.commands.Complete(ctx, current)
		},
	)
}
