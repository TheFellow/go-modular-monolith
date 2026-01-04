package menu

import (
	"github.com/TheFellow/go-modular-monolith/app/menu/authz"
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) RemoveDrink(ctx *middleware.Context, change models.MenuDrinkChange) (models.Menu, error) {
	return middleware.RunCommand(ctx, authz.ActionRemoveDrink, m.commands.RemoveDrink, change)
}
