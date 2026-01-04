package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Set(ctx *middleware.Context, update models.StockUpdate) (models.Stock, error) {
	return middleware.RunCommand(ctx, authz.ActionSet, m.commands.Set, update)
}
