package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Set(ctx *middleware.Context, update models.StockUpdate) (models.Stock, error) {
	if update.Quantity < 0 {
		update.Quantity = 0
	}
	if err := update.CostPerUnit.Validate(); err != nil {
		return models.Stock{}, err
	}
	if c.ingredients == nil {
		return models.Stock{}, errors.Internalf("missing ingredients dependency")
	}

	ingredient, err := c.ingredients.Get(ctx, update.IngredientID)
	if err != nil {
		return models.Stock{}, err
	}
	if ingredient.Unit == "" {
		return models.Stock{}, errors.Invalidf("ingredient unit is required")
	}

	ingredientIDStr := string(update.IngredientID.ID)
	existing, found, err := c.dao.Get(ctx, update.IngredientID)
	if err != nil {
		return models.Stock{}, errors.Internalf("get stock %s: %w", ingredientIDStr, err)
	}
	if !found {
		existing = models.Stock{
			IngredientID: update.IngredientID,
			Quantity:     0,
			Unit:         ingredient.Unit,
			LastUpdated:  time.Time{},
		}
	}

	previousQty := existing.Quantity
	newQty := update.Quantity
	delta := newQty - previousQty

	existing.Quantity = newQty
	existing.Unit = ingredient.Unit
	existing.CostPerUnit = optional.Some(update.CostPerUnit)
	existing.LastUpdated = time.Now().UTC()

	if err := c.dao.Upsert(ctx, existing); err != nil {
		return models.Stock{}, err
	}

	ctx.AddEvent(events.StockAdjusted{
		IngredientID: update.IngredientID,
		PreviousQty:  previousQty,
		NewQty:       newQty,
		Delta:        delta,
		Reason:       "set",
	})

	return existing, nil
}
