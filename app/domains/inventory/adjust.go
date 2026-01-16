package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Adjust(ctx *middleware.Context, patch models.Patch) (*models.Inventory, error) {
	return middleware.RunCommand(ctx, authz.ActionAdjust,
		middleware.FromModel(&models.Inventory{IngredientID: patch.IngredientID}),
		func(ctx *middleware.Context, _ *models.Inventory) (*models.Inventory, error) {
			return m.commands.Adjust(ctx, patch)
		},
	)
}
