package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) Delete(ctx *middleware.Context, id cedar.EntityUID) (*models.Ingredient, error) {
	return middleware.RunCommand(ctx, authz.ActionDelete,
		middleware.Get(m.queries.Get, id),
		m.commands.Delete,
	)
}
