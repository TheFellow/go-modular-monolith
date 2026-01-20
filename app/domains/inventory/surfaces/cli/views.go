package cli

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
)

type InventoryRow struct {
	IngredientID string `table:"INGREDIENT_ID" json:"ingredient_id"`
	Quantity     string `table:"QUANTITY" json:"quantity"`
	Unit         string `table:"UNIT" json:"unit"`
}

func ToInventoryRow(s *models.Inventory) InventoryRow {
	if s == nil {
		return InventoryRow{}
	}
	return InventoryRow{
		IngredientID: s.IngredientID.String(),
		Quantity:     fmt.Sprintf("%.2f", s.Amount.Value()),
		Unit:         string(s.Amount.Unit()),
	}
}

func ToInventoryRows(items []*models.Inventory) []InventoryRow {
	rows := make([]InventoryRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, ToInventoryRow(item))
	}
	return rows
}
