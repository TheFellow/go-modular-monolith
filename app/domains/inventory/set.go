package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Set(ctx *middleware.Context, update *models.Update, edits ...tag.Edit) (*models.Inventory, error) {
	return m.pipeline.Command(ctx, authz.ActionSet, update, withTags(edits, m.commands.Set))
}
