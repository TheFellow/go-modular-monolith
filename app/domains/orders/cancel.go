package orders

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Cancel(ctx *middleware.Context, order models.Order) (*models.Order, error) {
	return middleware.RunCommand(ctx, authz.ActionCancel,
		middleware.ByID(order.ID, m.queries.Get),
		m.commands.Cancel,
	)
}
