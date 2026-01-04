package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Create(ctx *middleware.Context, drink models.Drink) (models.Drink, error) {
	return middleware.RunCommand(ctx, authz.ActionCreate, m.commands.Create, drink)
}
