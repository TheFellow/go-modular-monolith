package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Update(ctx *middleware.Context, ingredient models.Ingredient) (*models.Ingredient, error) {
	return middleware.RunCommand(ctx, authz.ActionUpdate,
		func(ctx *middleware.Context) (*models.Ingredient, error) {
			return m.queries.Get(ctx, ingredient.ID)
		},
		func(ctx *middleware.Context, current *models.Ingredient) (*models.Ingredient, error) {
			return m.commands.Update(ctx, current, &ingredient)
		},
	)
}
