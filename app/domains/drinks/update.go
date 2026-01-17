package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Update(ctx *middleware.Context, drink *models.Drink) (*models.Drink, error) {
	return middleware.RunCommand(ctx, authz.ActionUpdate,
		middleware.Get(m.queries.Get, drink.ID),
		middleware.Update(m.commands.Update, drink),
	)
}
