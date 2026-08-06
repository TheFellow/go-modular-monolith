package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Delete(ctx *middleware.Context, id entity.IngredientID) (*models.Ingredient, error) {
	return m.Retire(ctx, id, models.Retirement{})
}

func (m *Module) Retire(ctx *middleware.Context, id entity.IngredientID, retirement models.Retirement) (*models.Ingredient, error) {
	return middleware.RunCommand(m.pipeline, ctx, middleware.CommandSpec[commands.RetirementTarget, *models.Ingredient]{
		Action: authz.ActionDelete,
		Load: func(ctx *middleware.Context) (commands.RetirementTarget, error) {
			ingredient, err := m.queries.Get(ctx, id)
			return commands.RetirementTarget{Ingredient: ingredient, Retirement: retirement}, err
		},
		Handle: m.commands.Retire,
	})
}
