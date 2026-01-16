package orders

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Place(ctx *middleware.Context, order models.Order) (*models.Order, error) {
	return middleware.RunCommand(ctx, authz.ActionPlace,
		func(*middleware.Context) (*models.Order, error) {
			toPlace := order
			if toPlace.ID.Type == "" {
				toPlace.ID = models.NewOrderID(string(toPlace.ID.ID))
			}
			return &toPlace, nil
		},
		m.commands.Place,
	)
}
