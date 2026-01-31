package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Delete(ctx *middleware.Context, id entity.DrinkID) (*models.Drink, error) {
	return middleware.RunCommand(ctx, authz.ActionDelete,
		middleware.Get(m.queries.Get, id),
		m.commands.Delete,
	)
}
