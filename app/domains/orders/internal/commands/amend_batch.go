package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) AmendBatch(ctx *middleware.Context, selected []*models.Order, requests []models.Amendment) ([]*models.Order, error) {
	result := make([]*models.Order, 0, len(selected))
	changes := make([]events.OrderAmended, 0, len(selected))
	for i, order := range selected {
		updated, err := c.amend(ctx, order, requests[i], changes)
		if err != nil {
			return nil, err
		}
		changes = append(changes, events.OrderAmended{Before: *order, Order: *updated, Reason: requests[i].Reason})
		result = append(result, updated)
	}
	ctx.AddEvent(events.OrdersAmended{Changes: changes})
	return result, nil
}
