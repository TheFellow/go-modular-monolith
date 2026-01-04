package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) Adjust(ctx *middleware.Context, ingredientID cedar.EntityUID, delta float64, reason models.AdjustmentReason) (models.Stock, error) {
	resource := cedar.Entity{
		UID:        models.NewStockID(ingredientID),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	return middleware.RunCommand(ctx, authz.ActionAdjust, resource, func(mctx *middleware.Context, _ struct{}) (models.Stock, error) {
		stock, err := m.commands.Adjust(mctx, ingredientID, delta, reason)
		if err != nil {
			return models.Stock{}, err
		}
		return stock, nil
	}, struct{}{})
}
