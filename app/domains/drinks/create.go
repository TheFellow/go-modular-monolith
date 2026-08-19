package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Create(ctx *middleware.Context, drink *models.Drink) (*models.Drink, error) {
	return m.pipeline.Command(ctx, authz.ActionCreate, drink, m.commands.Create)
}
