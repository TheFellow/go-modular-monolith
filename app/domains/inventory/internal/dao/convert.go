package dao

import (
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
)

func toRow(s inventorymodels.Stock) StockRow {
	return StockRow{
		IngredientID: string(s.IngredientID.ID),
		Quantity:     s.Quantity,
		Unit:         string(s.Unit),
		CostPerUnit:  s.CostPerUnit,
		LastUpdated:  s.LastUpdated,
	}
}

func toModel(r StockRow) inventorymodels.Stock {
	return inventorymodels.Stock{
		IngredientID: entity.IngredientID(r.IngredientID),
		Quantity:     r.Quantity,
		Unit:         measurement.Unit(r.Unit),
		CostPerUnit:  r.CostPerUnit,
		LastUpdated:  r.LastUpdated,
	}
}
