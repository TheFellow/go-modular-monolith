package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) AddDrink(ctx *middleware.Context, change *models.MenuPatch) (*models.Menu, error) {
	return middleware.RunCommand(ctx, middleware.CommandSpec[*models.MenuPatch, *models.Menu]{
		Action: authz.ActionAddDrink,
		Load: func(*middleware.Context) (*models.MenuPatch, error) {
			return change, nil
		},
		Handle: m.commands.AddDrink,
	})
}
