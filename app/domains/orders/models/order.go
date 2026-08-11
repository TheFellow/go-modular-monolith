package models

import (
	"time"

	orderauthz "github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

const OrderEntityType = entity.TypeOrder

func NewOrderID(id string) entity.OrderID {
	return entity.OrderID(cedar.NewEntityUID(entity.TypeOrder, cedar.String(id)))
}

type Order struct {
	ID                 entity.OrderID
	Revision           uint64 `json:"revision"`
	MenuID             entity.MenuID
	Items              []OrderItem
	IngredientUsage    []IngredientUsage
	BlockedIngredients []entity.IngredientID
	Status             OrderStatus
	CreatedAt          time.Time
	CompletedAt        optional.Value[time.Time]
	Notes              string
	DeletedAt          optional.Value[time.Time]
	Tags               tag.Tags
}

func (o Order) EntityUID() cedar.EntityUID {
	return o.ID.EntityUID()
}

func (o *Order) SetTags(tags tag.Tags) { o.Tags = tags }

func (o Order) CedarEntity() cedar.Entity {
	return orderauthz.Order{
		UID: o.ID.EntityUID(), MenuID: o.MenuID.EntityUID(), Status: string(o.Status), Tags: o.Tags.Map(),
	}.CedarEntity()
}

func (o Order) Validate() error {
	if o.MenuID.IsZero() {
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
	return nil
}

type OrderItem struct {
	DrinkID  entity.DrinkID
	Quantity int
	Notes    string
}

// IngredientUsage is the fulfillment snapshot captured when an order is placed.
// It insulates an accepted order from later recipe edits and is the contract used
// by Inventory to reserve, consume, and release stock.
type IngredientUsage struct {
	IngredientID entity.IngredientID
	Name         string
	Amount       measurement.Amount
}

func (i OrderItem) Validate() error {
	if i.DrinkID.IsZero() {
		return errors.Invalidf("drink id is required")
	}
	if i.Quantity <= 0 {
		return errors.Invalidf("quantity must be > 0")
	}
	return nil
}

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusBlocked   OrderStatus = "blocked"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusCancelled OrderStatus = "cancelled"
)

func (s OrderStatus) Validate() error {
	switch s {
	case OrderStatusPending, OrderStatusBlocked, OrderStatusCompleted, OrderStatusCancelled:
		return nil
	default:
		return errors.Invalidf("invalid status %q", string(s))
	}
}
