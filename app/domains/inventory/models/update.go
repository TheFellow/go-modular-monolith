package models

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	cedar "github.com/cedar-policy/cedar-go"
)

type StockUpdate struct {
	IngredientID cedar.EntityUID
	Quantity     float64
	CostPerUnit  money.Price
}

func (u StockUpdate) EntityUID() cedar.EntityUID {
	return NewInventoryID(u.IngredientID)
}

func (u StockUpdate) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        u.EntityUID(),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}
