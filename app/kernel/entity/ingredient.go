package entity

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

const (
	TypeIngredient   = cedar.EntityType("Mixology::Ingredient")
	PrefixIngredient = "ing"
)

// IngredientID is a strongly typed ID for ingredient entities.
type IngredientID cedar.EntityUID

// NewIngredientID generates a new ingredient ID with a KSUID.
func NewIngredientID() IngredientID {
	return IngredientID(NewID(TypeIngredient, PrefixIngredient))
}

// ParseIngredientID creates an ingredient ID from a stored string.
func ParseIngredientID(id string) (IngredientID, error) {
	if id == "" {
		return IngredientID(cedar.NewEntityUID(TypeIngredient, cedar.String(""))), nil
	}
	if !strings.HasPrefix(id, PrefixIngredient+"-") {
		return IngredientID{}, errors.Invalidf("invalid ingredient id prefix: %s", id)
	}
	return IngredientID(cedar.NewEntityUID(TypeIngredient, cedar.String(id))), nil
}

// EntityUID converts the ID to a cedar.EntityUID.
func (id IngredientID) EntityUID() cedar.EntityUID {
	return cedar.EntityUID(id)
}

// String returns the string form of the ID.
func (id IngredientID) String() string {
	return string(cedar.EntityUID(id).ID)
}

// IsZero returns true when the ID is unset.
func (id IngredientID) IsZero() bool {
	return cedar.EntityUID(id).ID == ""
}
