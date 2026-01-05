package orders

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type GetRequest struct {
	ID cedar.EntityUID
}

type GetResponse struct {
	Order models.Order
}

func (m *Module) Get(ctx *middleware.Context, req GetRequest) (GetResponse, error) {
	return middleware.RunQueryWithResource(ctx, authz.ActionGet, m.get, req)
}

func (m *Module) get(ctx *middleware.Context, req GetRequest) (GetResponse, error) {
	o, err := m.queries.Get(ctx, req.ID)
	if err != nil {
		return GetResponse{}, err
	}
	return GetResponse{Order: o}, nil
}

func (r GetRequest) CedarEntity() cedar.Entity {
	uid := r.ID
	if string(uid.ID) == "" {
		uid = cedar.NewEntityUID(models.OrderEntityType, cedar.String(""))
	}
	return cedar.Entity{
		UID:        uid,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}
