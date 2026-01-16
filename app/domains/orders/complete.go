package orders

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Complete(ctx *middleware.Context, order models.Order) (*models.Order, error) {
	return middleware.RunCommand(ctx, authz.ActionComplete,
		func(ctx *middleware.Context) (*models.Order, error) {
			return m.queries.Get(ctx, order.ID)
		},
		m.commands.Complete,
	)
}
