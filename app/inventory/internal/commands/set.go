package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Set(ctx *middleware.Context, update models.StockUpdate) (models.Stock, error) {
	if string(update.IngredientID.ID) == "" {
		return models.Stock{}, errors.Invalidf("ingredient id is required")
	}
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

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Stock{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Stock{}, errors.Internalf("register dao: %w", err)
	}

	ingredientIDStr := string(update.IngredientID.ID)
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
	newQty := update.Quantity
	delta := newQty - previousQty

	existing.Quantity = newQty
	existing.Unit = string(ingredient.Unit)
	existing.CostPerUnit = &update.CostPerUnit
	existing.LastUpdated = time.Now().UTC()

	if err := c.dao.Set(ctx, existing); err != nil {
		return models.Stock{}, err
	}

	ctx.AddEvent(events.StockAdjusted{
		IngredientID: update.IngredientID,
		PreviousQty:  previousQty,
		NewQty:       newQty,
		Delta:        delta,
		Reason:       "set",
	})

	return existing.ToDomain(), nil
}
