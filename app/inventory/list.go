package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type ListRequest struct{}

type ListResponse struct {
	Stock []models.Stock
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) (ListResponse, error) {
	return middleware.RunQuery(ctx, authz.ActionList, func(mctx *middleware.Context, _ ListRequest) (ListResponse, error) {
		stock, err := m.queries.List(mctx)
		if err != nil {
			return ListResponse{}, err
		}
		return ListResponse{Stock: stock}, nil
	}, req)
}
