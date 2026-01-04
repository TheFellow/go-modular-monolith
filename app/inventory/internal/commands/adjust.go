package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/money"
)

func (c *Commands) Adjust(ctx *middleware.Context, patch models.StockPatch) (models.Stock, error) {
	if string(patch.IngredientID.ID) == "" {
		return models.Stock{}, errors.Invalidf("ingredient id is required")
	}
	if patch.Reason == "" {
		return models.Stock{}, errors.Invalidf("reason is required")
	}

	var (
		hasDelta bool
		delta    float64
	)
	if v, ok := patch.Delta.Unwrap(); ok {
		hasDelta = true
		delta = v
	}

	var (
		hasCost bool
		cost    money.Price
	)
	if v, ok := patch.CostPerUnit.Unwrap(); ok {
		if err := v.Validate(); err != nil {
			return models.Stock{}, err
		}
		hasCost = true
		cost = v
	}

	if !hasDelta && !hasCost {
		return models.Stock{}, errors.Invalidf("at least one of delta or cost_per_unit is required")
	}

	if c.ingredients == nil {
		return models.Stock{}, errors.Internalf("missing ingredients dependency")
	}

	ingredient, err := c.ingredients.Get(ctx, patch.IngredientID)
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

	ingredientIDStr := string(patch.IngredientID.ID)
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
	newQty := previousQty
	appliedDelta := 0.0
	if hasDelta {
		newQty = previousQty + delta
		if newQty < 0 {
			newQty = 0
		}
		appliedDelta = newQty - previousQty
	}

	if hasDelta {
		existing.Quantity = newQty
	}
	existing.Unit = string(ingredient.Unit)
	if hasCost {
		existing.CostPerUnit = &cost
	}
	existing.LastUpdated = time.Now().UTC()

	if err := c.dao.Set(ctx, existing); err != nil {
		return models.Stock{}, err
	}

	if hasDelta {
		ctx.AddEvent(events.StockAdjusted{
			IngredientID: patch.IngredientID,
			PreviousQty:  previousQty,
			NewQty:       newQty,
			Delta:        appliedDelta,
			Reason:       string(patch.Reason),
		})
	}

	return existing.ToDomain(), nil
}
