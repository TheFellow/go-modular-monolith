package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type GetRequest struct {
	IngredientID cedar.EntityUID
}

type GetResponse struct {
	Stock models.Stock
}

func (m *Module) Get(ctx *middleware.Context, req GetRequest) (GetResponse, error) {
	return middleware.RunQuery(ctx, authz.ActionGet, func(mctx *middleware.Context, req GetRequest) (GetResponse, error) {
		s, err := m.queries.Get(mctx, req.IngredientID)
		if err != nil {
			return GetResponse{}, err
		}
		return GetResponse{Stock: s}, nil
	}, req)
}
