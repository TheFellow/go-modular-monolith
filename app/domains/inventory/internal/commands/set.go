package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Set(ctx *middleware.Context, update models.StockUpdate) (*models.Stock, error) {
	if update.Quantity < 0 {
		update.Quantity = 0
	}
	if err := update.CostPerUnit.Validate(); err != nil {
		return nil, err
	}
	if c.ingredients == nil {
		return nil, errors.Internalf("missing ingredients dependency")
	}

	ingredient, err := c.ingredients.Get(ctx, update.IngredientID)
	if err != nil {
		return nil, err
	}
	if ingredient.Unit == "" {
		return nil, errors.Invalidf("ingredient unit is required")
	}

	existing, err := c.dao.Get(ctx, update.IngredientID)
	var current models.Stock
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		current = models.Stock{
			IngredientID: update.IngredientID,
			Quantity:     0,
			Unit:         ingredient.Unit,
			LastUpdated:  time.Time{},
		}
	} else {
		current = *existing
	}

	previous := current

	current.Quantity = update.Quantity
	current.Unit = ingredient.Unit
	current.CostPerUnit = optional.Some(update.CostPerUnit)
	current.LastUpdated = time.Now().UTC()

	if err := c.dao.Upsert(ctx, current); err != nil {
		return nil, err
	}

	ctx.AddEvent(events.StockAdjusted{
		Previous: previous,
		Current:  current,
		Reason:   "set",
	})

	return &current, nil
}
