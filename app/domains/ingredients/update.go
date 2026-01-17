package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Update(ctx *middleware.Context, ingredient *models.Ingredient) (*models.Ingredient, error) {
	return middleware.RunCommand(ctx, authz.ActionUpdate,
		middleware.Get(m.queries.Get, ingredient.ID),
		middleware.Update(m.commands.Update, ingredient),
	)
}
