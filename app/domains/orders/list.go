package orders

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	ordersdao "github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type ListRequest struct {
	Status models.OrderStatus
	MenuID cedar.EntityUID
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

func (ListRequest) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        cedar.NewEntityUID(models.OrderEntityType, cedar.String("")),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}
