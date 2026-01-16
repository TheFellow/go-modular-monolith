package drinks

import (
	"reflect"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Update(ctx *middleware.Context, drink models.Drink) (*models.Drink, error) {
	return middleware.RunCommand(ctx, authz.ActionUpdate,
		func(ctx *middleware.Context) (models.Drink, error) {
			current, err := m.queries.Get(ctx, drink.ID)
			if err != nil {
				return models.Drink{}, err
			}
			return *current, nil
		},
		func(ctx *middleware.Context, current models.Drink) (*models.Drink, error) {
			updated, err := m.commands.Update(ctx, drink)
			if err != nil {
				return nil, err
			}
			if !reflect.DeepEqual(current.Recipe, updated.Recipe) {
				ctx.AddEvent(events.DrinkRecipeUpdated{
					Previous: current,
					Current:  *updated,
				})
			}
			return updated, nil
		},
	)
}
