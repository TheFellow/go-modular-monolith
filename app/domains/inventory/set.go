package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Set(ctx *middleware.Context, update models.Update) (*models.Inventory, error) {
	return middleware.RunCommand(ctx, authz.ActionSet,
		func(ctx *middleware.Context) (*models.Inventory, error) {
			return m.loadInventory(ctx, update.IngredientID)
		},
		func(ctx *middleware.Context, current *models.Inventory) (*models.Inventory, error) {
			return m.commands.Set(ctx, current, update)
		},
	)
}
