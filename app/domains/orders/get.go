package orders

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Get(ctx *middleware.Context, id entity.OrderID) (*models.Order, error) {
	return middleware.RunEntityQuery(m.pipeline, ctx, authz.ActionGet, m.queries.Get, id)
}
