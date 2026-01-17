package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Set(ctx *middleware.Context, update *models.Update) (*models.Inventory, error) {
	return middleware.RunCommand(ctx, authz.ActionSet,
		middleware.Entity(update),
		m.commands.Set,
	)
}
