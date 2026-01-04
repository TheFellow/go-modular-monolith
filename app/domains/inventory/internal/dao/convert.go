package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
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
		IngredientID: models.NewIngredientID(r.IngredientID),
		Quantity:     r.Quantity,
		Unit:         models.Unit(r.Unit),
		CostPerUnit:  r.CostPerUnit,
		LastUpdated:  r.LastUpdated,
	}
}
