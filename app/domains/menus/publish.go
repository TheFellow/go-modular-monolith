package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Publish(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
	return middleware.RunCommand(ctx, middleware.CommandSpec[*models.Menu, *models.Menu]{
		Action: authz.ActionPublish,
		Load: func(ctx *middleware.Context) (*models.Menu, error) {
			return m.queries.Get(ctx, menu.ID)
		},
		Handle: m.commands.Publish,
	})
}
