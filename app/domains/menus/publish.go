package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Publish(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
	if menu == nil {
		return nil, errors.Invalidf("menu is required")
	}
	return m.pipeline.LoadCommand(ctx, authz.ActionPublish,
		func(ctx *middleware.Context) (*models.Menu, error) {
			loaded, err := m.queries.Get(ctx, menu.ID)
			if err != nil {
				return nil, err
			}
			if menu.Revision != 0 && menu.Revision != loaded.Revision {
				return nil, errors.Conflictf("menu changed; reload before publish")
			}
			return loaded, nil
		},
		m.commands.Publish,
	)
}
