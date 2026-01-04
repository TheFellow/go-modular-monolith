package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Adjust(ctx *middleware.Context, patch models.StockPatch) (models.Stock, error) {
	return middleware.RunCommand(ctx, authz.ActionAdjust, m.commands.Adjust, patch)
}
