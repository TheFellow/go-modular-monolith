package events

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	cedar "github.com/cedar-policy/cedar-go"
)

type OrderPlaced struct {
	OrderID cedar.EntityUID
	MenuID  cedar.EntityUID
	Items   []OrderItemPlaced
	At      time.Time
	Notes   string
}

type OrderItemPlaced struct {
	DrinkID   cedar.EntityUID
	Quantity  int
	ItemNotes string
}

type OrderCancelled struct {
	OrderID cedar.EntityUID
	MenuID  cedar.EntityUID
	At      time.Time
}

type OrderCompleted struct {
	OrderID cedar.EntityUID
	MenuID  cedar.EntityUID
	Items   []OrderItemCompleted

	IngredientUsage     []IngredientUsage
	DepletedIngredients []cedar.EntityUID

	At time.Time
}

type OrderItemCompleted struct {
	DrinkID   cedar.EntityUID
	Name      string
	Quantity  int
	ItemNotes string
}

type IngredientUsage struct {
	IngredientID cedar.EntityUID
	Name         string
	Amount       float64
	Unit         string
}

func OrderPlacedFromDomain(o models.Order) OrderPlaced {
	items := make([]OrderItemPlaced, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, OrderItemPlaced{
			DrinkID:   it.DrinkID,
			Quantity:  it.Quantity,
			ItemNotes: it.Notes,
		})
	}
	return OrderPlaced{
		OrderID: o.ID,
		MenuID:  o.MenuID,
		Items:   items,
		At:      o.CreatedAt,
		Notes:   o.Notes,
	}
}
