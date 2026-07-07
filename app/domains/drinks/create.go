package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Create(ctx *middleware.Context, drink *models.Drink) (*models.Drink, error) {
	return middleware.RunCommand(ctx, middleware.CommandSpec[*models.Drink, *models.Drink]{
		Action: authz.ActionCreate,
		Load: func(*middleware.Context) (*models.Drink, error) {
			return drink, nil
		},
		Handle: m.commands.Create,
	})
}
