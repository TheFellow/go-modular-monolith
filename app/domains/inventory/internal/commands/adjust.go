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

func (c *Commands) Adjust(ctx *middleware.Context, patch *models.Patch) (*models.Inventory, error) {
	if patch == nil {
		return nil, errors.Invalidf("patch is required")
	}
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
	var updated models.Inventory
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		updated = models.Inventory{
			IngredientID: patch.IngredientID,
			Quantity:     0,
			Unit:         ingredient.Unit,
			CostPerUnit:  optional.None[money.Price](),
			LastUpdated:  time.Time{},
		}
	} else {
		updated = *existing
	}

	previousQty := updated.Quantity
	newQty := previousQty
	if hasDelta {
		newQty = previousQty + delta
		if newQty < 0 {
			newQty = 0
		}
	}

	if hasDelta {
		updated.Quantity = newQty
	}
	updated.IngredientID = patch.IngredientID
	updated.Unit = ingredient.Unit
	if hasCost {
		updated.CostPerUnit = optional.Some(cost)
	}
	updated.LastUpdated = time.Now().UTC()

	if err := c.dao.Upsert(ctx, updated); err != nil {
		return nil, err
	}

	ctx.TouchEntity(updated.EntityUID())
	if hasDelta {
		ctx.AddEvent(events.StockAdjusted{
			Inventory: updated,
			Reason:    string(patch.Reason),
		})
	}

	return &updated, nil
}
