package ingredients

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type GetRequest struct {
	ID string
}

type GetResponse struct {
	Ingredient models.Ingredient
}

func (m *Module) Get(ctx context.Context, req GetRequest) (GetResponse, error) {
	return middleware.RunQuery(ctx, authz.ActionGet, func(mctx *middleware.Context, req GetRequest) (GetResponse, error) {
		i, err := m.queries.Get(mctx, req.ID)
		if err != nil {
			return GetResponse{}, err
		}
		return GetResponse{Ingredient: i}, nil
	}, req)
}
