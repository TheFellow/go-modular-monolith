package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/cedar-policy/cedar-go"
)

type CreateRequest struct {
	Name string
}

type CreateResponse struct {
	Drink models.Drink
}

func (m *Module) Create(ctx *middleware.Context, req CreateRequest) (CreateResponse, error) {
	resource := cedar.Entity{
		UID:        cedar.NewEntityUID(cedar.EntityType("Mixology::Drink"), cedar.String("")),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	return middleware.RunCommand(ctx, authz.ActionCreate, resource, func(mctx *middleware.Context, req CreateRequest) (CreateResponse, error) {
		d, err := m.create.Execute(mctx, req.Name)
		if err != nil {
			return CreateResponse{}, err
		}
		return CreateResponse{Drink: d}, nil
	}, req)
}
