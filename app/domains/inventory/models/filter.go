package models

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/filter"
)

type ListFilterView struct {
	ID           string    `expr:"id" filter:"Inventory ID" filter-column:"InventoryID"`
	IngredientID string    `expr:"ingredient_id" filter:"Ingredient ID" filter-column:"IngredientID"`
	Quantity     float64   `expr:"quantity" filter:"Quantity on hand" filter-column:"Quantity"`
	Unit         string    `expr:"unit" filter:"Measurement unit" filter-column:"Unit"`
	LastUpdated  time.Time `expr:"last_updated" filter:"Last update timestamp" filter-column:"LastUpdated"`
}

func ListFilterSchema() filter.Schema[ListFilterView] {
	return filter.NewSchema[ListFilterView](
		`quantity <= 5 && unit == "ml"`,
		`ingredient_id.startsWith("ing-") || quantity == 0`,
	)
}
