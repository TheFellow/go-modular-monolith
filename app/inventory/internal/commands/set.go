package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Set(ctx *middleware.Context, stock models.Stock) (models.Stock, error) {
	if string(stock.IngredientID.ID) == "" {
		return models.Stock{}, errors.Invalidf("ingredient id is required")
	}
	if stock.Quantity < 0 {
		stock.Quantity = 0
	}
	if c.ingredients == nil {
		return models.Stock{}, errors.Internalf("missing ingredients dependency")
	}

	ingredient, err := c.ingredients.Get(ctx, stock.IngredientID)
	if err != nil {
		return models.Stock{}, err
	}
	if ingredient.Unit == "" {
		return models.Stock{}, errors.Invalidf("ingredient unit is required")
	}
	if stock.CostPerUnit != nil {
		if err := stock.CostPerUnit.Validate(); err != nil {
			return models.Stock{}, err
		}
	}

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Stock{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Stock{}, errors.Internalf("register dao: %w", err)
	}

	ingredientIDStr := string(stock.IngredientID.ID)
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
	newQty := stock.Quantity
	delta := newQty - previousQty

	existing.Quantity = newQty
	existing.Unit = string(ingredient.Unit)
	if stock.CostPerUnit != nil {
		existing.CostPerUnit = stock.CostPerUnit
	}
	existing.LastUpdated = time.Now().UTC()

	if err := c.dao.Set(ctx, existing); err != nil {
		return models.Stock{}, err
	}

	ctx.AddEvent(events.StockAdjusted{
		IngredientID: stock.IngredientID,
		PreviousQty:  previousQty,
		NewQty:       newQty,
		Delta:        delta,
		Reason:       "set",
	})

	return existing.ToDomain(), nil
}
