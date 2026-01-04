package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/inventory/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type SetRequest struct {
	IngredientID cedar.EntityUID
	Quantity     float64
}

type SetResponse struct {
	Stock models.Stock
}

func (m *Module) Set(ctx *middleware.Context, req SetRequest) (SetResponse, error) {
	ingredient, err := m.ingredientsQueries.Get(ctx, req.IngredientID)
	if err != nil {
		return SetResponse{}, err
	}

	resource := cedar.Entity{
		UID:        models.NewStockID(req.IngredientID),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	return middleware.RunCommand(ctx, authz.ActionSet, resource, func(mctx *middleware.Context, req SetRequest) (SetResponse, error) {
		stock, err := m.set.Execute(mctx, commands.SetRequest{
			IngredientID: req.IngredientID,
			Quantity:     req.Quantity,
			Unit:         ingredient.Unit,
		})
		if err != nil {
			return SetResponse{}, err
		}
		return SetResponse{Stock: stock}, nil
	}, req)
}
