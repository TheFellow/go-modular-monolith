package menus

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	menudao "github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type ListRequest struct {
	Status models.MenuStatus // Optional filter
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) ([]*models.Menu, error) {
	return middleware.RunQuery(ctx, authz.ActionList, m.list, req)
}

func (m *Module) list(ctx store.Context, req ListRequest) ([]*models.Menu, error) {
	menus, err := m.queries.List(ctx, menudao.ListFilter{Status: req.Status})
	if err != nil {
		return nil, err
	}
	return menus, nil
}

func (m *Module) Count(ctx *middleware.Context, req ListRequest) (int, error) {
	return middleware.RunQuery(ctx, authz.ActionList, m.count, req)
}

func (m *Module) count(ctx store.Context, req ListRequest) (int, error) {
	return m.queries.Count(ctx, menudao.ListFilter{Status: req.Status})
}
