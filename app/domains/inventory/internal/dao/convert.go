package dao

import (
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

func toRow(s inventorymodels.Inventory) StockRow {
	var costPerUnit *money.Price
	if cost, ok := s.CostPerUnit.Unwrap(); ok {
		costPerUnit = &cost
	}
	return StockRow{
		IngredientID: s.IngredientID.String(),
		InventoryID:  s.ID.String(),
		Quantity:     s.Amount.Value(),
		Unit:         string(s.Amount.Unit()),
		CostPerUnit:  costPerUnit,
		LastUpdated:  s.LastUpdated,
	}
}

func toModel(r StockRow) inventorymodels.Inventory {
	var costPerUnit optional.Value[money.Price]
	if r.CostPerUnit != nil {
		costPerUnit = optional.Some(*r.CostPerUnit)
	} else {
		costPerUnit = optional.None[money.Price]()
	}
	return inventorymodels.Inventory{
		ID:           entity.InventoryID(cedar.NewEntityUID(entity.TypeInventory, cedar.String(r.InventoryID))),
		IngredientID: entity.IngredientID(cedar.NewEntityUID(entity.TypeIngredient, cedar.String(r.IngredientID))),
		Amount:       measurement.MustAmount(r.Quantity, measurement.Unit(r.Unit)),
		CostPerUnit:  costPerUnit,
		LastUpdated:  r.LastUpdated,
	}
}
