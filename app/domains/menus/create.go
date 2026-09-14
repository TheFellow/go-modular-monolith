package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Create(ctx *middleware.Context, menu *models.Menu, edits ...tag.Edit) (*models.Menu, error) {
	return m.pipeline.Command(ctx, authz.ActionCreate, menu, withTags(edits, m.commands.Create))
}
