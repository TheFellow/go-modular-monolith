package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Update(ctx *middleware.Context, drink models.Drink) (*models.Drink, error) {
	return middleware.RunCommand(ctx, authz.ActionUpdate,
		middleware.ByID(drink.ID, m.queries.Get),
		func(ctx *middleware.Context, _ *models.Drink) (*models.Drink, error) {
			return m.commands.Update(ctx, &drink)
		},
	)
}
