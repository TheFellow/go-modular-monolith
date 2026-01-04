package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type UpdateRecipeRequest struct {
	ID     cedar.EntityUID
	Recipe models.Recipe
}

type UpdateRecipeResponse struct {
	Drink models.Drink
}

func (m *Module) UpdateRecipe(ctx *middleware.Context, req UpdateRecipeRequest) (UpdateRecipeResponse, error) {
	resource := cedar.Entity{
		UID:        req.ID,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	return middleware.RunCommand(ctx, authz.ActionUpdateRecipe, resource, func(mctx *middleware.Context, req UpdateRecipeRequest) (UpdateRecipeResponse, error) {
		d, err := m.commands.UpdateRecipe(mctx, commands.UpdateRecipeRequest{DrinkID: req.ID, Recipe: req.Recipe})
		if err != nil {
			return UpdateRecipeResponse{}, err
		}
		return UpdateRecipeResponse{Drink: d}, nil
	}, req)
}
