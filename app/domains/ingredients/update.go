package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Update(ctx *middleware.Context, ingredient models.Ingredient) (*models.Ingredient, error) {
	return middleware.RunCommand(ctx, authz.ActionUpdate,
		middleware.ByID(ingredient.ID, m.queries.Get),
		func(ctx *middleware.Context, _ *models.Ingredient) (*models.Ingredient, error) {
			return m.commands.Update(ctx, &ingredient)
		},
	)
}
