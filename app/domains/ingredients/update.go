package ingredients

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Update(ctx *middleware.Context, ingredient models.Ingredient) (*models.Ingredient, error) {
	return middleware.RunCommand(ctx, authz.ActionUpdate,
		func(ctx *middleware.Context) (*models.Ingredient, error) {
			return m.queries.Get(ctx, ingredient.ID)
		},
		func(ctx *middleware.Context, current *models.Ingredient) (*models.Ingredient, error) {
			previous := *current
			updated := previous
			if name := strings.TrimSpace(ingredient.Name); name != "" {
				updated.Name = name
			}
			if ingredient.Category != "" {
				updated.Category = ingredient.Category
			}
			if ingredient.Unit != "" {
				updated.Unit = ingredient.Unit
			}
			if desc := strings.TrimSpace(ingredient.Description); desc != "" {
				updated.Description = desc
			}

			result, err := m.commands.Update(ctx, &updated)
			if err != nil {
				return nil, err
			}
			ctx.AddEvent(events.IngredientUpdated{
				Previous: previous,
				Current:  *result,
			})
			return result, nil
		},
	)
}
