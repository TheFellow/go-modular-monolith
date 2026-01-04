package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Update(ctx *middleware.Context, ingredient models.Ingredient) (models.Ingredient, error) {
	return middleware.RunCommand(ctx, authz.ActionUpdate, m.commands.Update, ingredient)
}
