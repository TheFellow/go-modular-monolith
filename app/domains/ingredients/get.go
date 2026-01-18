package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) Get(ctx *middleware.Context, id cedar.EntityUID) (*models.Ingredient, error) {
	return middleware.RunQuery(ctx, authz.ActionGet, m.queries.Get, id)
}
