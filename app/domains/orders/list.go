package orders

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	ordersdao "github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

type ListRequest struct {
	Status models.OrderStatus
	MenuID entity.MenuID
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) ([]*models.Order, error) {
	return middleware.RunQueryWithResource(ctx, authz.ActionList, m.list, req)
}

func (m *Module) list(ctx store.Context, req ListRequest) ([]*models.Order, error) {
	os, err := m.queries.List(ctx, ordersdao.ListFilter{Status: req.Status, MenuID: req.MenuID})
	if err != nil {
		return nil, err
	}
	return os, nil
}

func (m *Module) Count(ctx *middleware.Context, req ListRequest) (int, error) {
	return middleware.RunQueryWithResource(ctx, authz.ActionList, m.count, req)
}

func (m *Module) count(ctx store.Context, req ListRequest) (int, error) {
	return m.queries.Count(ctx, ordersdao.ListFilter{Status: req.Status, MenuID: req.MenuID})
}

func (ListRequest) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        cedar.NewEntityUID(models.OrderEntityType, cedar.String("")),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}
