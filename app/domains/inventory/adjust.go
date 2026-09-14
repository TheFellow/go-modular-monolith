package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Adjust(ctx *middleware.Context, patch *models.Patch, edits ...tag.Edit) (*models.Inventory, error) {
	return m.pipeline.Command(ctx, authz.ActionAdjust, patch, withTags(edits, m.commands.Adjust))
}
