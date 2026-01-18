package entity

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

const (
	TypeOrder   = cedar.EntityType("Mixology::Order")
	PrefixOrder = "ord"
)

// OrderID is a strongly typed ID for order entities.
type OrderID cedar.EntityUID

// NewOrderID generates a new order ID with a KSUID.
func NewOrderID() OrderID {
	return OrderID(NewID(TypeOrder, PrefixOrder))
}

// ParseOrderID creates an order ID from a stored string.
func ParseOrderID(id string) (OrderID, error) {
	if id == "" {
		return OrderID(cedar.NewEntityUID(TypeOrder, cedar.String(""))), nil
	}
	if !strings.HasPrefix(id, PrefixOrder+"-") {
		return OrderID{}, errors.Invalidf("invalid order id prefix: %s", id)
	}
	return OrderID(cedar.NewEntityUID(TypeOrder, cedar.String(id))), nil
}

// EntityUID converts the ID to a cedar.EntityUID.
func (id OrderID) EntityUID() cedar.EntityUID {
	return cedar.EntityUID(id)
}

// String returns the string form of the ID.
func (id OrderID) String() string {
	return string(cedar.EntityUID(id).ID)
}

// IsZero returns true when the ID is unset.
func (id OrderID) IsZero() bool {
	return cedar.EntityUID(id).ID == ""
}
