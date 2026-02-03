package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Draft(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
	return middleware.RunCommand(ctx, authz.ActionDraft,
		middleware.Get(m.queries.Get, menu.ID),
		m.commands.Draft,
	)
}
