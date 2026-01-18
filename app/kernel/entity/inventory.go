package entity

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

const (
	TypeInventory   = cedar.EntityType("Mixology::Inventory")
	PrefixInventory = "inv"
)

// InventoryID is a strongly typed ID for inventory entities.
type InventoryID cedar.EntityUID

// NewInventoryID creates an inventory ID from an ingredient ID.
func NewInventoryID(ingredientID IngredientID) InventoryID {
	return InventoryID(cedar.NewEntityUID(TypeInventory, cedar.EntityUID(ingredientID).ID))
}

// ParseInventoryID creates an inventory ID from a stored string.
func ParseInventoryID(id string) (InventoryID, error) {
	if id == "" {
		return InventoryID(cedar.NewEntityUID(TypeInventory, cedar.String(""))), nil
	}
	if !strings.HasPrefix(id, PrefixIngredient+"-") {
		return InventoryID{}, errors.Invalidf("invalid inventory ingredient id prefix: %s", id)
	}
	return InventoryID(cedar.NewEntityUID(TypeInventory, cedar.String(id))), nil
}

// EntityUID converts the ID to a cedar.EntityUID.
func (id InventoryID) EntityUID() cedar.EntityUID {
	return cedar.EntityUID(id)
}

// String returns the string form of the ID.
func (id InventoryID) String() string {
	return string(cedar.EntityUID(id).ID)
}

// IsZero returns true when the ID is unset.
func (id InventoryID) IsZero() bool {
	return cedar.EntityUID(id).ID == ""
}
