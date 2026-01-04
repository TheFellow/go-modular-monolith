package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type AdjustParams struct {
	IngredientID cedar.EntityUID
	Delta        float64
	Reason       models.AdjustmentReason
}

func (p AdjustParams) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        models.NewStockID(p.IngredientID),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}

func (c *Commands) Adjust(ctx *middleware.Context, params AdjustParams) (models.Stock, error) {
	if string(params.IngredientID.ID) == "" {
		return models.Stock{}, errors.Invalidf("ingredient id is required")
	}
	if params.Reason == "" {
		return models.Stock{}, errors.Invalidf("reason is required")
	}
	if c.ingredients == nil {
		return models.Stock{}, errors.Internalf("missing ingredients dependency")
	}

	ingredient, err := c.ingredients.Get(ctx, params.IngredientID)
	if err != nil {
		return models.Stock{}, err
	}
	if ingredient.Unit == "" {
		return models.Stock{}, errors.Invalidf("ingredient unit is required")
	}

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Stock{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Stock{}, errors.Internalf("register dao: %w", err)
	}

	ingredientIDStr := string(params.IngredientID.ID)
	existing, found, err := c.dao.Get(ctx, ingredientIDStr)
	if err != nil {
		return models.Stock{}, errors.Internalf("get stock %s: %w", ingredientIDStr, err)
	}
	if !found {
		existing = dao.Stock{
			IngredientID: ingredientIDStr,
			Quantity:     0,
			Unit:         string(ingredient.Unit),
			LastUpdated:  time.Time{},
		}
	}

	previousQty := existing.Quantity
	newQty := previousQty + params.Delta
	if newQty < 0 {
		newQty = 0
	}
	appliedDelta := newQty - previousQty

	existing.Quantity = newQty
	existing.Unit = string(ingredient.Unit)
	existing.LastUpdated = time.Now().UTC()

	if err := c.dao.Set(ctx, existing); err != nil {
		return models.Stock{}, err
	}

	ctx.AddEvent(events.StockAdjusted{
		IngredientID: params.IngredientID,
		PreviousQty:  previousQty,
		NewQty:       newQty,
		Delta:        appliedDelta,
		Reason:       string(params.Reason),
	})

	return existing.ToDomain(), nil
}
