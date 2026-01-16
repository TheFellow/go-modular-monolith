package menu

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Publish(ctx *middleware.Context, menu models.Menu) (*models.Menu, error) {
	return middleware.RunCommand(ctx, authz.ActionPublish,
		func(ctx *middleware.Context) (models.Menu, error) {
			current, err := m.queries.Get(ctx, menu.ID)
			if err != nil {
				return models.Menu{}, err
			}
			return *current, nil
		},
		func(ctx *middleware.Context, current models.Menu) (*models.Menu, error) {
			return m.commands.Publish(ctx, current)
		},
	)
}
