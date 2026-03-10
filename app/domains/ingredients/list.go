package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	ingredientsdao "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type ListRequest struct {
	Category models.Category
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) ([]*models.Ingredient, error) {
	return middleware.RunQuery(ctx, authz.ActionList, m.list, req)
}

func (m *Module) list(ctx store.Context, req ListRequest) ([]*models.Ingredient, error) {
	is, err := m.queries.List(ctx, ingredientsdao.ListFilter{Category: req.Category})
	if err != nil {
		return nil, err
	}
	return is, nil
}

func (m *Module) Count(ctx *middleware.Context, req ListRequest) (int, error) {
	return middleware.RunQuery(ctx, authz.ActionList, m.count, req)
}

func (m *Module) count(ctx store.Context, req ListRequest) (int, error) {
	return m.queries.Count(ctx, ingredientsdao.ListFilter{Category: req.Category})
}
