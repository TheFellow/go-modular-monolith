package dao

import (
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	cedar "github.com/cedar-policy/cedar-go"
)

func toRow(s inventorymodels.Inventory) StockRow {
	return StockRow{
		IngredientID: s.IngredientID.String(),
		Quantity:     s.Amount.Value(),
		Unit:         string(s.Amount.Unit()),
		CostPerUnit:  s.CostPerUnit,
		LastUpdated:  s.LastUpdated,
	}
}

func toModel(r StockRow) inventorymodels.Inventory {
	return inventorymodels.Inventory{
		IngredientID: entity.IngredientID(cedar.NewEntityUID(entity.TypeIngredient, cedar.String(r.IngredientID))),
		Amount:       measurement.MustAmount(r.Quantity, measurement.Unit(r.Unit)),
		CostPerUnit:  r.CostPerUnit,
		LastUpdated:  r.LastUpdated,
	}
}
