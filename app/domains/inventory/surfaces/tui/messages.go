package tui

import (
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
)

type InventoryRow struct {
	Inventory  inventorymodels.Inventory
	Ingredient ingredientsmodels.Ingredient
	Quantity   string
	Cost       string
	Status     string
}

// InventoryLoadedMsg is sent when inventory has been loaded.
type InventoryLoadedMsg struct {
	Rows []InventoryRow
	Err  error
}

// InventoryAdjustedMsg is sent when inventory is adjusted.
type InventoryAdjustedMsg struct {
	Inventory *inventorymodels.Inventory
}

// InventorySetMsg is sent when inventory is set.
type InventorySetMsg struct {
	Inventory *inventorymodels.Inventory
}
