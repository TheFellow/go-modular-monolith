package dao

import (
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
)

func toRow(o models.Order) OrderRow {
	items := make([]OrderItemRow, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, OrderItemRow{
			DrinkID:  it.DrinkID,
			Quantity: it.Quantity,
			Notes:    it.Notes,
		})
	}

	return OrderRow{
		ID:          string(o.ID.ID),
		MenuID:      string(o.MenuID.ID),
		Items:       items,
		Status:      string(o.Status),
		CreatedAt:   o.CreatedAt,
		CompletedAt: o.CompletedAt,
		Notes:       o.Notes,
	}
}

func toModel(r OrderRow) models.Order {
	items := make([]models.OrderItem, 0, len(r.Items))
	for _, it := range r.Items {
		items = append(items, models.OrderItem{
			DrinkID:  it.DrinkID,
			Quantity: it.Quantity,
			Notes:    it.Notes,
		})
	}

	return models.Order{
		ID:          models.NewOrderID(r.ID),
		MenuID:      menumodels.NewMenuID(r.MenuID),
		Items:       items,
		Status:      models.OrderStatus(r.Status),
		CreatedAt:   r.CreatedAt,
		CompletedAt: r.CompletedAt,
		Notes:       r.Notes,
	}
}
