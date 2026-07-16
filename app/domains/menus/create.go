package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Create(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
	return middleware.RunCommand(m.pipeline, ctx, middleware.CommandSpec[*models.Menu, *models.Menu]{
		Action: authz.ActionCreate,
		Load: func(*middleware.Context) (*models.Menu, error) {
			return menu, nil
		},
		Handle: m.commands.Create,
	})
}
