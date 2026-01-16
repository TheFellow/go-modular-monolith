package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Adjust(ctx *middleware.Context, patch models.Patch) (*models.Inventory, error) {
	return middleware.RunCommand(ctx, authz.ActionAdjust,
		func(ctx *middleware.Context) (*models.Inventory, error) {
			return m.loadInventory(ctx, patch.IngredientID)
		},
		func(ctx *middleware.Context, current *models.Inventory) (*models.Inventory, error) {
			return m.commands.Adjust(ctx, current, patch)
		},
	)
}
