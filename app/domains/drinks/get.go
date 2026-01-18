package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/cedar-policy/cedar-go"
)

func (m *Module) Get(ctx *middleware.Context, id cedar.EntityUID) (*models.Drink, error) {
	return middleware.RunQuery(ctx, authz.ActionGet, m.queries.Get, id)
}
