package entity

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

const (
	TypeDrink   = cedar.EntityType("Mixology::Drink")
	PrefixDrink = "drk"
)

// DrinkID is a strongly typed ID for drink entities.
type DrinkID cedar.EntityUID

// NewDrinkID generates a new drink ID with a KSUID.
func NewDrinkID() DrinkID {
	return DrinkID(NewID(TypeDrink, PrefixDrink))
}

// ParseDrinkID creates a drink ID from a stored string.
func ParseDrinkID(id string) (DrinkID, error) {
	if id == "" {
		return DrinkID(cedar.NewEntityUID(TypeDrink, cedar.String(""))), nil
	}
	if !strings.HasPrefix(id, PrefixDrink+"-") {
		return DrinkID{}, errors.Invalidf("invalid drink id prefix: %s", id)
	}
	return DrinkID(cedar.NewEntityUID(TypeDrink, cedar.String(id))), nil
}

// EntityUID converts the ID to a cedar.EntityUID.
func (id DrinkID) EntityUID() cedar.EntityUID {
	return cedar.EntityUID(id)
}

// String returns the string form of the ID.
func (id DrinkID) String() string {
	return string(cedar.EntityUID(id).ID)
}

// IsZero returns true when the ID is unset.
func (id DrinkID) IsZero() bool {
	return cedar.EntityUID(id).ID == ""
}
