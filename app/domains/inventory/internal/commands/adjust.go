package commands

import (
	"fmt"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
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
		delta    measurement.Amount
	)
	if v, ok := patch.Delta.Unwrap(); ok {
		if v == nil {
			return nil, errors.Invalidf("delta is required")
		}
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
			ID:           entity.NewInventoryID(),
			IngredientID: patch.IngredientID,
			Amount:       measurement.MustAmount(0, ingredient.Unit),
			CostPerUnit:  optional.None[money.Price](),
			LastUpdated:  time.Time{},
		}
	} else {
		if patch.Revision != 0 && patch.Revision != existing.Revision {
			return nil, errors.Conflictf("stock changed: reload before editing")
		}
		if existing.Status != "" && existing.Status != models.StatusActive {
			return nil, errors.FailedPreconditionf("stock is %s; release quarantine before editing active stock", existing.Status)
		}
		updated = *existing
	}
	if updated.ID.IsZero() {
		updated.ID = entity.NewInventoryID()
	}

	updatedAmount, err := updated.Amount.Convert(ingredient.Unit)
	if err != nil {
		return nil, err
	}
	if hasDelta {
		delta, err = delta.Convert(ingredient.Unit)
		if err != nil {
			return nil, err
		}
		updatedAmount, err = updatedAmount.Add(delta)
		if err != nil {
			return nil, err
		}
		if updatedAmount.Value() < 0 {
			updatedAmount = measurement.MustAmount(0, ingredient.Unit)
		}
	}
	updated.Amount = updatedAmount
	updated.IngredientID = patch.IngredientID
	if hasCost {
		updated.CostPerUnit = optional.Some(cost)
		updated.CostUnit = patch.CostUnit
		if updated.CostUnit == "" {
			updated.CostUnit = ingredient.Unit
		}
		if _, err := measurement.MustAmount(1, ingredient.Unit).Convert(updated.CostUnit); err != nil {
			return nil, err
		}
	}
	updated.IngredientName = ingredient.Name
	updated.LastUpdated = time.Now().UTC()
	updated.Reason = string(patch.Reason)
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
	if hasDelta {
		reserved, err := c.dao.ReservedAmount(ctx, updated.IngredientID)
		if err != nil {
			return nil, err
		}
		ctx.AddEvent(events.StockAdjusted{
			Inventory: updated,
			Reason:    string(patch.Reason),
			Shortage:  updated.Amount.Value() < reserved.Value(),
		})
	}

	return &updated, nil
}
