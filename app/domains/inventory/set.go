package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Set(ctx *middleware.Context, update models.StockUpdate) (models.Stock, error) {
	return middleware.RunCommand(ctx, authz.ActionSet, m.commands.Set, update)
}
