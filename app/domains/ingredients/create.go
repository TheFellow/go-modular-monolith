package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Create(ctx *middleware.Context, ingredient *models.Ingredient) (*models.Ingredient, error) {
	return m.pipeline.Command(ctx, authz.ActionCreate, ingredient, m.commands.Create)
}
