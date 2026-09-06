package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Dispose(ctx *middleware.Context, request models.Disposal) (*models.Inventory, error) {
	return m.pipeline.LoadCommand(ctx, authz.ActionAdjust,
		func(ctx *middleware.Context) (*models.Inventory, error) {
			return m.queries.Get(ctx, request.IngredientID)
		},
		func(ctx *middleware.Context, stock *models.Inventory) (*models.Inventory, error) {
			return m.commands.Dispose(ctx, stock, request)
		})
}
func (m *Module) History(ctx *middleware.Context, id entity.IngredientID) ([]models.Movement, error) {
	stock, err := m.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return m.queries.History(ctx, stock.ID)
}

func (m *Module) Disposition(ctx *middleware.Context, request models.Disposition) (*models.Inventory, error) {
	return m.pipeline.LoadCommand(ctx, authz.ActionAdjust,
		func(ctx *middleware.Context) (*models.Inventory, error) {
			return m.queries.Get(ctx, request.IngredientID)
		},
		func(ctx *middleware.Context, stock *models.Inventory) (*models.Inventory, error) {
			return m.commands.Disposition(ctx, stock, request)
		})
}
