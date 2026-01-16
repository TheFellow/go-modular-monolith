package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Create(ctx *middleware.Context, ingredient models.Ingredient) (*models.Ingredient, error) {
	return middleware.RunCommand(ctx, authz.ActionCreate,
		func(*middleware.Context) (*models.Ingredient, error) {
			toCreate := ingredient
			if toCreate.ID.Type == "" {
				toCreate.ID = entity.IngredientID(string(toCreate.ID.ID))
			}
			return &toCreate, nil
		},
		m.commands.Create,
	)
}
