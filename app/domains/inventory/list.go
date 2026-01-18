package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	inventorydao "github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

type ListRequest struct {
	IngredientID cedar.EntityUID

	// LowStock, when set, lists items with Quantity <= LowStock.
	LowStock optional.Value[float64]
}

func (ListRequest) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        cedar.NewEntityUID(models.InventoryEntityType, cedar.String("")),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) ([]*models.Inventory, error) {
	return middleware.RunQueryWithResource(ctx, authz.ActionList, m.list, req)
}

func (m *Module) list(ctx store.Context, req ListRequest) ([]*models.Inventory, error) {
	filter := inventorydao.ListFilter{
		IngredientID: req.IngredientID,
		MaxQuantity:  req.LowStock,
	}
	stock, err := m.queries.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	return stock, nil
}
