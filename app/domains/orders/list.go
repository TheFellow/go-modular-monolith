package orders

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type ListRequest struct{}

type ListResponse struct {
	Orders []models.Order
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) (ListResponse, error) {
	return middleware.RunQueryWithResource(ctx, authz.ActionList, m.list, req)
}

func (m *Module) list(ctx *middleware.Context, _ ListRequest) (ListResponse, error) {
	os, err := m.queries.List(ctx)
	if err != nil {
		return ListResponse{}, err
	}
	return ListResponse{Orders: os}, nil
}

func (ListRequest) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        cedar.NewEntityUID(models.OrderEntityType, cedar.String("")),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}
