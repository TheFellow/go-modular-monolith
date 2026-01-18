package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type getRequest struct {
	IngredientID cedar.EntityUID
}

func (r getRequest) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        models.NewInventoryID(r.IngredientID),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}

func (m *Module) Get(ctx *middleware.Context, ingredientID cedar.EntityUID) (*models.Inventory, error) {
	return middleware.RunQueryWithResource(ctx, authz.ActionGet, m.get, getRequest{IngredientID: ingredientID})
}

func (m *Module) get(ctx dao.Context, req getRequest) (*models.Inventory, error) {
	return m.queries.Get(ctx, req.IngredientID)
}
