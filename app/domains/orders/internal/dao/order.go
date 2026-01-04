package dao

import (
	"time"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

type Order struct {
	ID          string      `json:"id"`
	MenuID      string      `json:"menu_id"`
	Items       []OrderItem `json:"items"`
	Status      string      `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	Notes       string      `json:"notes,omitempty"`
}

type OrderItem struct {
	DrinkID  string `json:"drink_id"`
	Quantity int    `json:"quantity"`
	Notes    string `json:"notes,omitempty"`
}

func (o Order) ToDomain() models.Order {
	completed := optional.NewNone[time.Time]()
	if o.CompletedAt != nil {
		completed = optional.NewSome(*o.CompletedAt)
	}

	items := make([]models.OrderItem, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, models.OrderItem{
			DrinkID:  drinksmodels.NewDrinkID(it.DrinkID),
			Quantity: it.Quantity,
			Notes:    it.Notes,
		})
	}

	return models.Order{
		ID:          models.NewOrderID(o.ID),
		MenuID:      menumodels.NewMenuID(o.MenuID),
		Items:       items,
		Status:      models.OrderStatus(o.Status),
		CreatedAt:   o.CreatedAt,
		CompletedAt: completed,
		Notes:       o.Notes,
	}
}

func FromDomain(o models.Order) Order {
	var completedAt *time.Time
	if t, ok := o.CompletedAt.Unwrap(); ok {
		completedAt = &t
	}

	items := make([]OrderItem, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, OrderItem{
			DrinkID:  string(it.DrinkID.ID),
			Quantity: it.Quantity,
			Notes:    it.Notes,
		})
	}

	return Order{
		ID:          string(o.ID.ID),
		MenuID:      string(o.MenuID.ID),
		Items:       items,
		Status:      string(o.Status),
		CreatedAt:   o.CreatedAt,
		CompletedAt: completedAt,
		Notes:       o.Notes,
	}
}
