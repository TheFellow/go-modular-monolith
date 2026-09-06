package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"strings"
	"time"
)

// Disposition changes whether retained stock can fulfill commitments. Releasing
// retired stock restores existing commitments without reopening the catalog.
func (c *Commands) Disposition(ctx *middleware.Context, stock *models.Inventory, request models.Disposition) (*models.Inventory, error) {
	if request.Revision == 0 || request.Revision != stock.Revision {
		return nil, errors.Conflictf("stock changed; reload before changing disposition")
	}
	if strings.TrimSpace(request.Reason) == "" {
		return nil, errors.Invalidf("a disposition reason is required")
	}
	updated := *stock
	if request.Quarantine {
		if stock.Status != models.StatusActive && stock.Status != models.StatusDiscontinued {
			return nil, errors.FailedPreconditionf("cannot quarantine %s stock", stock.Status)
		}
		updated.Status = models.StatusQuarantined
	} else {
		if stock.Status != models.StatusQuarantined {
			return nil, errors.FailedPreconditionf("only quarantined stock can be released")
		}
		_, err := c.ingredients.Get(ctx, stock.IngredientID)
		switch {
		case errors.IsNotFound(err):
			updated.Status = models.StatusDiscontinued
		case err != nil:
			return nil, err
		default:
			updated.Status = models.StatusActive
		}
	}
	updated.Reason = request.Reason
	updated.LastUpdated = time.Now().UTC()
	if err := c.dao.Upsert(ctx, &updated); err != nil {
		return nil, err
	}
	shortage, err := updated.Amount.LessThan(updated.ReservedAmount())
	if err != nil {
		return nil, err
	}
	ctx.RecordEffect("stock_disposition_changed", stock.EntityUID(), middleware.Change("status", stock.Status, updated.Status), middleware.Change("reason", "", request.Reason))
	ctx.AddEvent(events.StockAdjusted{Inventory: updated, Reason: request.Reason, Shortage: request.Quarantine || shortage})
	return &updated, nil
}
