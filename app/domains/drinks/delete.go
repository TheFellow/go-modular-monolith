package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) Delete(ctx *middleware.Context, id cedar.EntityUID) (*models.Drink, error) {
	return middleware.RunCommand(ctx, authz.ActionDelete,
		func(ctx *middleware.Context) (*models.Drink, error) {
			return m.queries.Get(ctx, id)
		},
		m.commands.Delete,
	)
}
