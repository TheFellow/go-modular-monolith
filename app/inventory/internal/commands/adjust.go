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

func (c *Commands) Adjust(ctx *middleware.Context, ingredientID cedar.EntityUID, delta float64, reason models.AdjustmentReason) (models.Stock, error) {
	if string(ingredientID.ID) == "" {
		return models.Stock{}, errors.Invalidf("ingredient id is required")
	}
	if reason == "" {
		return models.Stock{}, errors.Invalidf("reason is required")
	}
	if c.ingredients == nil {
		return models.Stock{}, errors.Internalf("missing ingredients dependency")
	}

	ingredient, err := c.ingredients.Get(ctx, ingredientID)
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

	ingredientIDStr := string(ingredientID.ID)
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
	newQty := previousQty + delta
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
		IngredientID: ingredientID,
		PreviousQty:  previousQty,
		NewQty:       newQty,
		Delta:        appliedDelta,
		Reason:       string(reason),
	})

	return existing.ToDomain(), nil
}
