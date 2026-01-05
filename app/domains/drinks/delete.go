package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type DeleteRequest struct {
	ID cedar.EntityUID
}

func (r DeleteRequest) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        r.ID,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}

type DeleteResponse struct {
	Drink models.Drink
}

func (m *Module) Delete(ctx *middleware.Context, req DeleteRequest) (DeleteResponse, error) {
	return middleware.RunCommand(ctx, authz.ActionDelete, m.delete, req)
}

func (m *Module) delete(ctx *middleware.Context, req DeleteRequest) (DeleteResponse, error) {
	drink, err := m.commands.Delete(ctx, req.ID)
	if err != nil {
		return DeleteResponse{}, err
	}
	return DeleteResponse{Drink: drink}, nil
}
