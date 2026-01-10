package dao

import (
	"time"

	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func toRow(o models.Order) OrderRow {
	var deletedAt *time.Time
	if t, ok := o.DeletedAt.Unwrap(); ok {
		deletedAt = &t
	}
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
		DeletedAt:   deletedAt,
	}
}

func toModel(r OrderRow) models.Order {
	var deletedAt optional.Value[time.Time]
	if r.DeletedAt != nil {
		deletedAt = optional.Some(*r.DeletedAt)
	} else {
		deletedAt = optional.None[time.Time]()
	}
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
		DeletedAt:   deletedAt,
	}
}
