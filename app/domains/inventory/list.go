package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	inventorydao "github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type ListRequest struct {
	IngredientID entity.IngredientID

	// LowStock, when set, lists items with amount value <= LowStock (per item unit).
	LowStock optional.Value[float64]
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) ([]*models.Inventory, error) {
	return middleware.RunListQuery(m.pipeline, ctx, authz.ActionList, m.list, req)
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

func (m *Module) Count(ctx *middleware.Context, req ListRequest) (int, error) {
	stock, err := m.List(ctx, req)
	if err != nil {
		return 0, err
	}
	return len(stock), nil
}
