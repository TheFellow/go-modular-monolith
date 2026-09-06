package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"strings"
	"time"
)

func (c *Commands) Dispose(ctx *middleware.Context, stock *models.Inventory, request models.Disposal) (*models.Inventory, error) {
	if request.Revision == 0 || request.Revision != stock.Revision {
		return nil, errors.Conflictf("stock changed; reload before disposal")
	}
	if stock.Status == models.StatusActive || stock.Status == "" {
		return nil, errors.FailedPreconditionf("discontinue or quarantine the ingredient before disposing stock")
	}
	if strings.TrimSpace(request.Reason) == "" || request.Amount == nil || request.Amount.Value() <= 0 {
		return nil, errors.Invalidf("positive disposal amount and reason are required")
	}
	updated := *stock
	amount, err := request.Amount.Convert(stock.Amount.Unit())
	if err != nil {
		return nil, err
	}
	updated.Amount, err = stock.Amount.Sub(amount)
	if err != nil {
		return nil, err
	}
	if updated.Amount.Value() < 0 {
		return nil, errors.Invalidf("disposal exceeds physical stock")
	}
	if updated.Amount.IsZero() {
		updated.Status = models.StatusDisposed
	}
	updated.Reason = request.Reason
	updated.LastUpdated = time.Now().UTC()
	if err := c.dao.Upsert(ctx, &updated); err != nil {
		return nil, err
	}
	shortage, err := updated.Amount.LessThan(stock.ReservedAmount())
	if err != nil {
		return nil, err
	}
	ctx.RecordEffect("stock_disposed", stock.EntityUID(), middlewareevents.Change{Field: "quantity", Before: stock.Amount.String(), After: updated.Amount.String()}, middlewareevents.Change{Field: "reason", After: request.Reason})
	ctx.AddEvent(events.StockAdjusted{Inventory: updated, Reason: request.Reason, Shortage: shortage || updated.Status == models.StatusQuarantined || updated.Status == models.StatusDisposed})
	return &updated, nil
}
