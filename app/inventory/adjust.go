package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/inventory/authz"
	inventorycommands "github.com/TheFellow/go-modular-monolith/app/inventory/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) Adjust(ctx *middleware.Context, ingredientID cedar.EntityUID, delta float64, reason models.AdjustmentReason) (models.Stock, error) {
	return middleware.RunCommand(ctx, authz.ActionAdjust, m.commands.Adjust, inventorycommands.AdjustParams{
		IngredientID: ingredientID,
		Delta:        delta,
		Reason:       reason,
	})
}
