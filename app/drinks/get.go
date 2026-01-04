package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type GetRequest struct {
	ID cedar.EntityUID
}

type GetResponse struct {
	Drink models.Drink
}

func (m *Module) Get(ctx *middleware.Context, req GetRequest) (GetResponse, error) {
	return middleware.RunQuery(ctx, authz.ActionGet, m.get, req)
}

func (m *Module) get(ctx *middleware.Context, req GetRequest) (GetResponse, error) {
	d, err := m.queries.Get(ctx, req.ID)
	if err != nil {
		return GetResponse{}, err
	}
	return GetResponse{Drink: d}, nil
}
