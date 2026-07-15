package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Delete(ctx *middleware.Context, id entity.MenuID) (*models.Menu, error) {
	return middleware.RunCommand(m.pipeline, ctx, middleware.CommandSpec[*models.Menu, *models.Menu]{
		Action: authz.ActionDelete,
		Load: func(ctx *middleware.Context) (*models.Menu, error) {
			return m.queries.Get(ctx, id)
		},
		Handle: m.commands.Delete,
	})
}
