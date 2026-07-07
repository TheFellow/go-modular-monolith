package orders

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Place(ctx *middleware.Context, order *models.Order) (*models.Order, error) {
	return middleware.RunCommand(ctx, middleware.CommandSpec[*models.Order, *models.Order]{
		Action: authz.ActionPlace,
		Load: func(*middleware.Context) (*models.Order, error) {
			return order, nil
		},
		Handle: m.commands.Place,
	})
}
