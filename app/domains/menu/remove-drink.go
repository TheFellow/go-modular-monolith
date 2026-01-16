package menu

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) RemoveDrink(ctx *middleware.Context, change models.MenuDrinkChange) (*models.Menu, error) {
	return middleware.RunCommand(ctx, authz.ActionRemoveDrink,
		func(ctx *middleware.Context) (*models.Menu, error) {
			return m.queries.Get(ctx, change.MenuID)
		},
		func(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
			return m.commands.RemoveDrink(ctx, menu, change)
		},
	)
}
