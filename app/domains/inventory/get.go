package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type GetRequest struct {
	IngredientID cedar.EntityUID
}

type GetResponse struct {
	Stock models.Stock
}

func (r GetRequest) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        models.NewStockID(r.IngredientID),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}

func (m *Module) Get(ctx *middleware.Context, req GetRequest) (GetResponse, error) {
	return middleware.RunQueryWithResource(ctx, authz.ActionGet, m.get, req)
}

func (m *Module) get(ctx *middleware.Context, req GetRequest) (GetResponse, error) {
	s, err := m.queries.Get(ctx, req.IngredientID)
	if err != nil {
		return GetResponse{}, err
	}
	return GetResponse{Stock: s}, nil
}
