package menu

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type ListRequest struct {
	Status models.MenuStatus // Optional filter
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) ([]*models.Menu, error) {
	return middleware.RunQuery(ctx, authz.ActionList, m.list, req)
}

func (m *Module) list(ctx *middleware.Context, req ListRequest) ([]*models.Menu, error) {
	menus, err := m.queries.List(ctx, dao.ListFilter{Status: req.Status})
	if err != nil {
		return nil, err
	}
	return menus, nil
}
