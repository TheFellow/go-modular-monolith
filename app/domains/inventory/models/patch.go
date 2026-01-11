package models

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

type StockPatch struct {
	IngredientID cedar.EntityUID
	Reason       AdjustmentReason
	Delta        optional.Value[float64]
	CostPerUnit  optional.Value[money.Price]
}

func (p StockPatch) EntityUID() cedar.EntityUID {
	_ = p.Reason
	_ = p.Delta
	_ = p.CostPerUnit
	return NewInventoryID(p.IngredientID)
}

func (p StockPatch) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        p.EntityUID(),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}
