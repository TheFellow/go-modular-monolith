package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/cedar-policy/cedar-go"
)

func (m *Module) Update(ctx *middleware.Context, ingredient models.Ingredient) (models.Ingredient, error) {
	resource := cedar.Entity{
		UID:        ingredient.ID,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	return middleware.RunCommand(ctx, authz.ActionUpdate, resource, func(mctx *middleware.Context, ingredient models.Ingredient) (models.Ingredient, error) {
		i, err := m.commands.Update(mctx, ingredient)
		if err != nil {
			return models.Ingredient{}, err
		}
		return i, nil
	}, ingredient)
}
