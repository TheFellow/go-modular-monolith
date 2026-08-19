package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Delete(ctx *middleware.Context, id entity.MenuID) (*models.Menu, error) {
	return m.pipeline.LoadCommand(ctx, authz.ActionDelete,
		func(ctx *middleware.Context) (*models.Menu, error) {
			return m.queries.Get(ctx, id)
		},
		m.commands.Delete,
	)
}
