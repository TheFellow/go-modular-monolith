package menu

import (
	"github.com/TheFellow/go-modular-monolith/app/menu/authz"
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type ListRequest struct{}

type ListResponse struct {
	Menus []models.Menu
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) (ListResponse, error) {
	return middleware.RunQuery(ctx, authz.ActionList, func(mctx *middleware.Context, _ ListRequest) (ListResponse, error) {
		menus, err := m.queries.List(mctx)
		if err != nil {
			return ListResponse{}, err
		}
		return ListResponse{Menus: menus}, nil
	}, req)
}
