package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Create(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
	return m.pipeline.Command(ctx, authz.ActionCreate, menu, m.commands.Create)
}
