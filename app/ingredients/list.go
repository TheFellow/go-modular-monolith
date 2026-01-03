package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type ListRequest struct{}

type ListResponse struct {
	Ingredients []models.Ingredient
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) (ListResponse, error) {
	return middleware.RunQuery(ctx, authz.ActionList, func(mctx *middleware.Context, _ ListRequest) (ListResponse, error) {
		is, err := m.queries.List(mctx)
		if err != nil {
			return ListResponse{}, err
		}
		return ListResponse{Ingredients: is}, nil
	}, req)
}
