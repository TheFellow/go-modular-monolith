package menu

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Create(ctx *middleware.Context, menu models.Menu) (*models.Menu, error) {
	return middleware.RunCommand(ctx, authz.ActionCreate,
		func(*middleware.Context) (*models.Menu, error) {
			toCreate := menu
			if toCreate.ID.Type == "" {
				toCreate.ID = models.NewMenuID(string(toCreate.ID.ID))
			}
			return &toCreate, nil
		},
		m.commands.Create,
	)
}
