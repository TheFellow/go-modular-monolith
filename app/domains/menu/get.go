package menu

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) Get(ctx *middleware.Context, id cedar.EntityUID) (*models.Menu, error) {
	return middleware.RunQuery(ctx, authz.ActionGet, m.get, id)
}

func (m *Module) get(ctx *middleware.Context, id cedar.EntityUID) (*models.Menu, error) {
	return m.queries.Get(ctx, id)
}
