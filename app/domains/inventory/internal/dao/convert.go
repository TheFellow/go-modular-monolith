package dao

import (
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

func toRow(s inventorymodels.Inventory) (StockRow, error) {
	amount, err := s.Amount.Convert(s.Amount.Unit().Canonical())
	if err != nil {
		return StockRow{}, err
	}
	var costPerUnit *money.Price
	if cost, ok := s.CostPerUnit.Unwrap(); ok {
		costPerUnit = &cost
	}
	return StockRow{
		Status: string(s.Status), Reason: s.Reason,
		IngredientID:   s.IngredientID.String(),
		IngredientName: s.IngredientName,
		Revision:       s.Revision,
		InventoryID:    s.ID.String(),
		Quantity:       amount.Value(),
		DisplayUnit:    string(s.Amount.Unit()),
		CostUnit:       string(s.CostUnit),
		Unit:           string(amount.Unit()),
		CostPerUnit:    costPerUnit,
		LastUpdated:    s.LastUpdated,
	}, nil
}

func toModel(r StockRow) (inventorymodels.Inventory, error) {
	amount, err := measurement.NewAmount(r.Quantity, measurement.Unit(r.Unit))
	if err != nil {
		return inventorymodels.Inventory{}, err
	}
	if r.DisplayUnit != "" {
		amount, err = amount.Convert(measurement.Unit(r.DisplayUnit))
		if err != nil {
			return inventorymodels.Inventory{}, err
		}
	}
	var costPerUnit optional.Value[money.Price]
	if r.CostPerUnit != nil {
		costPerUnit = optional.Some(*r.CostPerUnit)
	} else {
		costPerUnit = optional.None[money.Price]()
	}
	return inventorymodels.Inventory{
		Status: inventorymodels.Status(r.Status), Reason: r.Reason,
		IngredientName: r.IngredientName,
		ID:             entity.InventoryID(cedar.NewEntityUID(entity.TypeInventory, cedar.String(r.InventoryID))),
		Revision:       r.Revision,
		IngredientID:   entity.IngredientID(cedar.NewEntityUID(entity.TypeIngredient, cedar.String(r.IngredientID))),
		Amount:         amount,
		CostUnit:       measurement.Unit(r.CostUnit),
		CostPerUnit:    costPerUnit,
		LastUpdated:    r.LastUpdated,
	}, nil
}
