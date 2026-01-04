package models

import (
	"github.com/TheFellow/go-modular-monolith/app/money"
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
	return NewStockID(p.IngredientID)
}

func (p StockPatch) CedarEntity() cedar.Entity {
	uid := p.EntityUID()
	if string(uid.ID) == "" {
		uid = cedar.NewEntityUID(StockEntityType, cedar.String(""))
	}
	return cedar.Entity{
		UID:        uid,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}
