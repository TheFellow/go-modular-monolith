package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Create(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
	return middleware.RunCommand(ctx, authz.ActionCreate,
		middleware.Entity(menu),
		m.commands.Create,
	)
}
