package models

import (
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

const OrderEntityType = cedar.EntityType("Mixology::Order")

func NewOrderID(id string) cedar.EntityUID {
	return cedar.NewEntityUID(OrderEntityType, cedar.String(id))
}

type Order struct {
	ID          string
	MenuID      cedar.EntityUID
	Items       []OrderItem
	Status      OrderStatus `bstore:"index"`
	CreatedAt   time.Time   `bstore:"index"`
	CompletedAt optional.Value[time.Time]
	Notes       string
}

func (o Order) EntityUID() cedar.EntityUID {
	return NewOrderID(o.ID)
}

func (o Order) CedarEntity() cedar.Entity {
	uid := NewOrderID(o.ID)
	return cedar.Entity{
		UID:        uid,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}

func (o Order) Validate() error {
	if string(o.MenuID.ID) == "" {
		return errors.Invalidf("menu id is required")
	}
	if err := o.Status.Validate(); err != nil {
		return err
	}
	if len(o.Items) == 0 {
		return errors.Invalidf("order must have at least 1 item")
	}
	for i := range o.Items {
		if err := o.Items[i].Validate(); err != nil {
			return errors.Invalidf("item %d: %w", i, err)
		}
	}
	o.Notes = strings.TrimSpace(o.Notes)
	return nil
}

type OrderItem struct {
	DrinkID  cedar.EntityUID
	Quantity int
	Notes    string
}

func (i OrderItem) Validate() error {
	if string(i.DrinkID.ID) == "" {
		return errors.Invalidf("drink id is required")
	}
	if i.Quantity <= 0 {
		return errors.Invalidf("quantity must be > 0")
	}
	i.Notes = strings.TrimSpace(i.Notes)
	return nil
}

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPreparing OrderStatus = "preparing"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusCancelled OrderStatus = "cancelled"
)

func (s OrderStatus) Validate() error {
	switch s {
	case OrderStatusPending, OrderStatusPreparing, OrderStatusCompleted, OrderStatusCancelled:
		return nil
	default:
		return errors.Invalidf("invalid status %q", string(s))
	}
}
