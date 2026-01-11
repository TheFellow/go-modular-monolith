package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type deleteRequest struct {
	ID cedar.EntityUID
}

func (r deleteRequest) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        r.ID,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}

func (m *Module) Delete(ctx *middleware.Context, id cedar.EntityUID) (*models.Ingredient, error) {
	return middleware.RunCommand(ctx, authz.ActionDelete, m.delete, deleteRequest{ID: id})
}

func (m *Module) delete(ctx *middleware.Context, req deleteRequest) (*models.Ingredient, error) {
	return m.commands.Delete(ctx, req.ID)
}
