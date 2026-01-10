package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Adjust(ctx *middleware.Context, patch models.StockPatch) (*models.Stock, error) {
	if patch.Reason == "" {
		return nil, errors.Invalidf("reason is required")
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
			return nil, err
		}
		hasCost = true
		cost = v
	}

	if !hasDelta && !hasCost {
		return nil, errors.Invalidf("at least one of delta or cost_per_unit is required")
	}

	if c.ingredients == nil {
		return nil, errors.Internalf("missing ingredients dependency")
	}

	ingredient, err := c.ingredients.Get(ctx, patch.IngredientID)
	if err != nil {
		return nil, err
	}
	if ingredient.Unit == "" {
		return nil, errors.Invalidf("ingredient unit is required")
	}

	existing, err := c.dao.Get(ctx, patch.IngredientID)
	var current models.Stock
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		current = models.Stock{
			IngredientID: patch.IngredientID,
			Quantity:     0,
			Unit:         ingredient.Unit,
			CostPerUnit:  optional.None[money.Price](),
			LastUpdated:  time.Time{},
		}
	} else {
		current = *existing
	}

	previous := current

	previousQty := current.Quantity
	newQty := previousQty
	if hasDelta {
		newQty = previousQty + delta
		if newQty < 0 {
			newQty = 0
		}
	}

	if hasDelta {
		current.Quantity = newQty
	}
	current.Unit = ingredient.Unit
	if hasCost {
		current.CostPerUnit = optional.Some(cost)
	}
	current.LastUpdated = time.Now().UTC()

	if err := c.dao.Upsert(ctx, current); err != nil {
		return nil, err
	}

	if hasDelta {
		ctx.AddEvent(events.StockAdjusted{
			Previous: previous,
			Current:  current,
			Reason:   string(patch.Reason),
		})
	}

	return &current, nil
}
