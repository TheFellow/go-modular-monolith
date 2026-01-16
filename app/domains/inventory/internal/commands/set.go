package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Set(ctx *middleware.Context, current *models.Inventory, update models.Update) (*models.Inventory, error) {
	if current == nil {
		return nil, errors.Internalf("inventory is required")
	}
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

	previous := *current
	updated := *current

	updated.IngredientID = update.IngredientID
	updated.Quantity = update.Quantity
	updated.Unit = ingredient.Unit
	updated.CostPerUnit = optional.Some(update.CostPerUnit)
	updated.LastUpdated = time.Now().UTC()

	if err := c.dao.Upsert(ctx, updated); err != nil {
		return nil, err
	}

	ctx.TouchEntity(updated.EntityUID())
	ctx.AddEvent(events.StockAdjusted{
		Previous: previous,
		Current:  updated,
		Reason:   "set",
	})

	return &updated, nil
}
