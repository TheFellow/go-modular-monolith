package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/inventory/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type AdjustRequest struct {
	IngredientID cedar.EntityUID
	Delta        float64
	Reason       models.AdjustmentReason
}

type AdjustResponse struct {
	Stock models.Stock
}

func (m *Module) Adjust(ctx *middleware.Context, req AdjustRequest) (AdjustResponse, error) {
	resource := cedar.Entity{
		UID:        models.NewStockID(req.IngredientID),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	return middleware.RunCommand(ctx, authz.ActionAdjust, resource, func(mctx *middleware.Context, req AdjustRequest) (AdjustResponse, error) {
		stock, err := m.commands.Adjust(mctx, commands.AdjustRequest{
			IngredientID: req.IngredientID,
			Delta:        req.Delta,
			Reason:       req.Reason,
		})
		if err != nil {
			return AdjustResponse{}, err
		}
		return AdjustResponse{Stock: stock}, nil
	}, req)
}
