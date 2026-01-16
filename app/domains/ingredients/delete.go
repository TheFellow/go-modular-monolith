package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) Delete(ctx *middleware.Context, id cedar.EntityUID) (*models.Ingredient, error) {
	return middleware.RunCommand(ctx, authz.ActionDelete,
		func(ctx *middleware.Context) (models.Ingredient, error) {
			loaded, err := m.queries.Get(ctx, id)
			if err != nil {
				return models.Ingredient{}, err
			}
			return *loaded, nil
		},
		func(ctx *middleware.Context, ingredient models.Ingredient) (*models.Ingredient, error) {
			return m.commands.Delete(ctx, ingredient)
		},
	)
}
