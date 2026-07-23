package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Update(ctx *middleware.Context, drink *models.Drink) (*models.Drink, error) {
	authorizedUpdate := middleware.AuthorizeCommand(authz.ActionUpdate, m.commands.Update)
	return middleware.RunCommand(m.pipeline, ctx, middleware.CommandSpec[*models.Drink, *models.Drink]{
		Action: authz.ActionUpdate,
		Load: func(ctx *middleware.Context) (*models.Drink, error) {
			return m.queries.Get(ctx, drink.ID)
		},
		Handle: func(ctx *middleware.Context, _ *models.Drink) (*models.Drink, error) {
			return authorizedUpdate(ctx, drink)
		},
	})
}
