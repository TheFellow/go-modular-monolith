package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/inventory/authz"
	inventorycommands "github.com/TheFellow/go-modular-monolith/app/inventory/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) Set(ctx *middleware.Context, ingredientID cedar.EntityUID, quantity float64) (models.Stock, error) {
	return middleware.RunCommand(ctx, authz.ActionSet, m.commands.Set, inventorycommands.SetParams{
		IngredientID: ingredientID,
		Quantity:     quantity,
	})
}
