package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Delete(ctx *middleware.Context, id entity.IngredientID) (*models.Ingredient, error) {
	return middleware.RunCommand(m.pipeline, ctx, middleware.CommandSpec[*models.Ingredient, *models.Ingredient]{
		Action: authz.ActionDelete,
		Load: func(ctx *middleware.Context) (*models.Ingredient, error) {
			return m.queries.Get(ctx, id)
		},
		Handle: m.commands.Delete,
	})
}
