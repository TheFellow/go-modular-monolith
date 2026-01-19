package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Set(ctx *middleware.Context, update *models.Update) (*models.Inventory, error) {
	if update == nil {
		return nil, errors.Invalidf("update is required")
	}
	if update.Amount == nil || update.Amount.Unit() == "" {
		return nil, errors.Invalidf("amount is required")
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
	var updated models.Inventory
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		updated = models.Inventory{
			IngredientID: update.IngredientID,
			Amount:       measurement.MustAmount(0, ingredient.Unit),
			LastUpdated:  time.Time{},
		}
	} else {
		updated = *existing
	}

	updated.IngredientID = update.IngredientID
	amount, err := update.Amount.Convert(ingredient.Unit)
	if err != nil {
		return nil, err
	}
	if amount.Value() < 0 {
		amount = measurement.MustAmount(0, ingredient.Unit)
	}
	updated.Amount = amount
	updated.CostPerUnit = optional.Some(update.CostPerUnit)
	updated.LastUpdated = time.Now().UTC()

	if err := c.dao.Upsert(ctx, updated); err != nil {
		return nil, err
	}

	ctx.TouchEntity(updated.EntityUID())
	ctx.AddEvent(events.StockAdjusted{
		Inventory: updated,
		Reason:    "set",
	})

	return &updated, nil
}
