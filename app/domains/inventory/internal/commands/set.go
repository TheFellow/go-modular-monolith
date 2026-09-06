package commands

import (
	"fmt"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
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
		if update.Revision != 0 {
			return nil, errors.Conflictf("stock no longer exists")
		}
		updated = models.Inventory{
			ID:           entity.NewInventoryID(),
			IngredientID: update.IngredientID,
			Amount:       measurement.MustAmount(0, ingredient.Unit),
			LastUpdated:  time.Time{},
		}
	} else {
		if update.Revision != existing.Revision {
			return nil, errors.Conflictf("stock changed: expected revision %d, current revision %d", update.Revision, existing.Revision)
		}
		if existing.Status != "" && existing.Status != models.StatusActive {
			return nil, errors.FailedPreconditionf("stock is %s; release quarantine before editing active stock", existing.Status)
		}
		updated = *existing
	}
	if updated.ID.IsZero() {
		updated.ID = entity.NewInventoryID()
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
	updated.CostUnit = update.CostUnit
	if updated.CostUnit == "" {
		updated.CostUnit = ingredient.Unit
	}
	if _, err := measurement.MustAmount(1, ingredient.Unit).Convert(updated.CostUnit); err != nil {
		return nil, err
	}
	updated.IngredientName = ingredient.Name
	updated.LastUpdated = time.Now().UTC()
	updated.Reason = "set"
	if updated.Status == "" {
		updated.Status = models.StatusActive
	}

	if err := c.dao.Upsert(ctx, &updated); err != nil {
		return nil, err
	}

	beforeAmount := "0"
	beforeCost := ""
	beforeCostUnit := measurement.Unit("")
	if existing != nil {
		beforeAmount = existing.Amount.String()
		beforeCost = fmt.Sprint(existing.CostPerUnit)
		beforeCostUnit = existing.CostUnit
	}
	ctx.RecordEffect("stock_adjusted", updated.EntityUID(), middleware.Change("quantity", beforeAmount, updated.Amount.String()), middleware.Change("cost", beforeCost, fmt.Sprint(updated.CostPerUnit)), middleware.Change("cost_unit", beforeCostUnit, updated.CostUnit), middleware.Change("reason", "", updated.Reason))
	reserved, err := c.dao.ReservedAmount(ctx, updated.IngredientID)
	if err != nil {
		return nil, err
	}
	ctx.AddEvent(events.StockAdjusted{
		Inventory: updated,
		Reason:    "set",
		Shortage:  updated.Amount.Value() < reserved.Value(),
	})

	return &updated, nil
}
