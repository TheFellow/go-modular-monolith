package entity

import "github.com/cedar-policy/cedar-go"

const TypeInventory = cedar.EntityType("Mixology::Inventory")

func InventoryID(ingredientID cedar.EntityUID) cedar.EntityUID {
	return cedar.NewEntityUID(TypeInventory, ingredientID.ID)
}
