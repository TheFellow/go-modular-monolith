package menu

import (
	"github.com/TheFellow/go-modular-monolith/app/menu/authz"
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type GetRequest struct {
	ID cedar.EntityUID
}

type GetResponse struct {
	Menu models.Menu
}

func (m *Module) Get(ctx *middleware.Context, req GetRequest) (GetResponse, error) {
	return middleware.RunQuery(ctx, authz.ActionGet, func(mctx *middleware.Context, req GetRequest) (GetResponse, error) {
		menu, err := m.queries.Get(mctx, req.ID)
		if err != nil {
			return GetResponse{}, err
		}
		return GetResponse{Menu: menu}, nil
	}, req)
}
