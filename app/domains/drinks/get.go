package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) Get(ctx *middleware.Context, id cedar.EntityUID) (*models.Drink, error) {
	return middleware.RunQuery(ctx, authz.ActionGet, m.get, id)
}

func (m *Module) get(ctx *middleware.Context, id cedar.EntityUID) (*models.Drink, error) {
	return m.queries.Get(ctx, id)
}
