package models

import (
	cedar "github.com/cedar-policy/cedar-go"
)

type StockAdjustment struct {
	IngredientID cedar.EntityUID
	Delta        float64
	Reason       AdjustmentReason
}

func (a StockAdjustment) EntityUID() cedar.EntityUID {
	return NewStockID(a.IngredientID)
}

func (a StockAdjustment) CedarEntity() cedar.Entity {
	uid := a.EntityUID()
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
