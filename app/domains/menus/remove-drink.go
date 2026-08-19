package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) RemoveDrink(ctx *middleware.Context, change *models.MenuPatch) (*models.Menu, error) {
	return m.pipeline.Command(ctx, authz.ActionRemoveDrink, change, m.commands.RemoveDrink)
}
