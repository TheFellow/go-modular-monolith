package menu

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) AddDrink(ctx *middleware.Context, change *models.MenuPatch) (*models.Menu, error) {
	return middleware.RunCommand(ctx, authz.ActionAddDrink,
		middleware.Entity(change),
		m.commands.AddDrink,
	)
}
