package orders

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

type getRequest struct {
	ID entity.OrderID
}

func (m *Module) Get(ctx *middleware.Context, id entity.OrderID) (*models.Order, error) {
	return middleware.RunQueryWithResource(ctx, authz.ActionGet, m.get, getRequest{ID: id})
}

func (m *Module) get(ctx store.Context, req getRequest) (*models.Order, error) {
	return m.queries.Get(ctx, req.ID)
}

func (r getRequest) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        r.ID.EntityUID(),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}
