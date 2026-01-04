package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) UpdateRecipe(ctx *middleware.Context, id cedar.EntityUID, recipe models.Recipe) (models.Drink, error) {
	resource := cedar.Entity{
		UID:        id,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	return middleware.RunCommand(ctx, authz.ActionUpdateRecipe, resource, func(mctx *middleware.Context, _ struct{}) (models.Drink, error) {
		d, err := m.commands.UpdateRecipe(mctx, id, recipe)
		if err != nil {
			return models.Drink{}, err
		}
		return d, nil
	}, struct{}{})
}
