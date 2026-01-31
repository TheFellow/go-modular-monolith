package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

type getRequest struct {
	IngredientID entity.IngredientID
}

func (r getRequest) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        cedar.NewEntityUID(models.InventoryEntityType, cedar.String("")),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}

func (m *Module) Get(ctx *middleware.Context, ingredientID entity.IngredientID) (*models.Inventory, error) {
	return middleware.RunQueryWithResource(ctx, authz.ActionGet, m.get, getRequest{IngredientID: ingredientID})
}

func (m *Module) get(ctx store.Context, req getRequest) (*models.Inventory, error) {
	return m.queries.Get(ctx, req.IngredientID)
}
