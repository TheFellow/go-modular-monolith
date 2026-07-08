package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Create(ctx *middleware.Context, ingredient *models.Ingredient) (*models.Ingredient, error) {
	return middleware.RunCommand(ctx, middleware.CommandSpec[*models.Ingredient, *models.Ingredient]{
		Action: authz.ActionCreate,
		Load: func(*middleware.Context) (*models.Ingredient, error) {
			return ingredient, nil
		},
		Handle: m.commands.Create,
	})
}
