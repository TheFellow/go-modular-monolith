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

type ListResponse struct {
	Ingredients []models.Ingredient
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) (ListResponse, error) {
	return middleware.RunQuery(ctx, authz.ActionList, m.list, req)
}

func (m *Module) list(ctx *middleware.Context, req ListRequest) (ListResponse, error) {
	is, err := m.queries.List(ctx, dao.ListFilter{Category: req.Category})
	if err != nil {
		return ListResponse{}, err
	}
	return ListResponse{Ingredients: is}, nil
}
