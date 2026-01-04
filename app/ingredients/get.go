package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type GetRequest struct {
	ID cedar.EntityUID
}

type GetResponse struct {
	Ingredient models.Ingredient
}

func (m *Module) Get(ctx *middleware.Context, req GetRequest) (GetResponse, error) {
	return middleware.RunQuery(ctx, authz.ActionGet, m.get, req)
}

func (m *Module) get(ctx *middleware.Context, req GetRequest) (GetResponse, error) {
	i, err := m.queries.Get(ctx, req.ID)
	if err != nil {
		return GetResponse{}, err
	}
	return GetResponse{Ingredient: i}, nil
}
