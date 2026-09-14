package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) RemoveDrink(ctx *middleware.Context, change *models.MenuPatch, edits ...tag.Edit) (*models.Menu, error) {
	return m.pipeline.Command(ctx, authz.ActionDrinkRemove, change, withTags(edits, m.commands.RemoveDrink))
}
