package menu

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Publish(ctx *middleware.Context, menu models.Menu) (*models.Menu, error) {
	return middleware.RunCommand(ctx, authz.ActionPublish,
		func(ctx *middleware.Context) (*models.Menu, error) {
			return m.queries.Get(ctx, menu.ID)
		},
		m.commands.Publish,
	)
}
