package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Update(ctx *middleware.Context, ingredient *models.Ingredient) (*models.Ingredient, error) {
	return middleware.RunCommand(m.pipeline, ctx, middleware.CommandSpec[*models.Ingredient, *models.Ingredient]{
		Action: authz.ActionUpdate,
		Load: func(ctx *middleware.Context) (*models.Ingredient, error) {
			return m.queries.Get(ctx, ingredient.ID)
		},
		Handle: func(ctx *middleware.Context, _ *models.Ingredient) (*models.Ingredient, error) {
			return m.commands.Update(ctx, ingredient)
		},
	})
}
