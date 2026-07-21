package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Get(ctx *middleware.Context, ingredientID entity.IngredientID) (*models.Inventory, error) {
	return middleware.RunEntityQuery(m.pipeline, ctx, authz.ActionGet, m.queries.Get, ingredientID)
}
