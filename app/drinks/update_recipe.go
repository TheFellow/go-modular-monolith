package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/drinks/authz"
	drinkscommands "github.com/TheFellow/go-modular-monolith/app/drinks/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) UpdateRecipe(ctx *middleware.Context, id cedar.EntityUID, recipe models.Recipe) (models.Drink, error) {
	return middleware.RunCommand(ctx, authz.ActionUpdateRecipe, m.commands.UpdateRecipe, drinkscommands.UpdateRecipeParams{
		DrinkID: id,
		Recipe:  recipe,
	})
}
