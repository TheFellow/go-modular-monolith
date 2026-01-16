package menu

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) AddDrink(ctx *middleware.Context, change models.MenuDrinkChange) (*models.Menu, error) {
	return middleware.RunCommand(ctx, authz.ActionAddDrink,
		middleware.ByID(change.MenuID, m.queries.Get),
		func(ctx *middleware.Context, _ *models.Menu) (*models.Menu, error) {
			return m.commands.AddDrink(ctx, change)
		},
	)
}
