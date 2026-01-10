package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type ListRequest struct {
	Category models.Category
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) ([]*models.Ingredient, error) {
	return middleware.RunQuery(ctx, authz.ActionList, m.list, req)
}

func (m *Module) list(ctx *middleware.Context, req ListRequest) ([]*models.Ingredient, error) {
	is, err := m.queries.List(ctx, dao.ListFilter{Category: req.Category})
	if err != nil {
		return nil, err
	}
	return is, nil
}
