package orders

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Amend(ctx *middleware.Context, request models.Amendment) (*models.Order, error) {
	return m.pipeline.LoadCommand(ctx, authz.ActionAmend,
		func(ctx *middleware.Context) (*models.Order, error) { return m.queries.Get(ctx, request.OrderID) },
		func(ctx *middleware.Context, order *models.Order) (*models.Order, error) {
			return m.commands.Amend(ctx, order, request)
		})
}
