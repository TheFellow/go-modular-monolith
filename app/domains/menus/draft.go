package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Draft(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
	return m.pipeline.LoadCommand(ctx, authz.ActionDraft,
		func(ctx *middleware.Context) (*models.Menu, error) {
			return m.queries.Get(ctx, menu.ID)
		},
		m.commands.Draft,
	)
}
