package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type ListRequest struct{}

type ListResponse struct {
	Drinks []models.Drink
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) (ListResponse, error) {
	return middleware.RunQuery(ctx, authz.ActionList, func(mctx *middleware.Context, _ ListRequest) (ListResponse, error) {
		ds, err := m.queries.List(mctx)
		if err != nil {
			return ListResponse{}, err
		}
		return ListResponse{Drinks: ds}, nil
	}, req)
}
