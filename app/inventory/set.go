package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) Set(ctx *middleware.Context, ingredientID cedar.EntityUID, quantity float64) (models.Stock, error) {
	resource := cedar.Entity{
		UID:        models.NewStockID(ingredientID),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	return middleware.RunCommand(ctx, authz.ActionSet, resource, func(mctx *middleware.Context, _ struct{}) (models.Stock, error) {
		stock, err := m.commands.Set(mctx, ingredientID, quantity)
		if err != nil {
			return models.Stock{}, err
		}
		return stock, nil
	}, struct{}{})
}
