package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Set(ctx *middleware.Context, current models.Inventory, update models.Update) (*models.Inventory, error) {
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

	previous := current

	current.IngredientID = update.IngredientID
	current.Quantity = update.Quantity
	current.Unit = ingredient.Unit
	current.CostPerUnit = optional.Some(update.CostPerUnit)
	current.LastUpdated = time.Now().UTC()

	if err := c.dao.Upsert(ctx, current); err != nil {
		return nil, err
	}

	ctx.TouchEntity(current.EntityUID())
	ctx.AddEvent(events.StockAdjusted{
		Previous: previous,
		Current:  current,
		Reason:   "set",
	})

	return &current, nil
}
