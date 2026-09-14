package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Create(ctx *middleware.Context, drink *models.Drink, edits ...tag.Edit) (*models.Drink, error) {
	return m.pipeline.Command(ctx, authz.ActionCreate, drink, withTags(edits, m.commands.Create))
}
