package models

import (
	"github.com/TheFellow/go-modular-monolith/app/money"
	cedar "github.com/cedar-policy/cedar-go"
)

type StockUpdate struct {
	IngredientID cedar.EntityUID
	Quantity     float64
	CostPerUnit  money.Price
}

func (u StockUpdate) EntityUID() cedar.EntityUID {
	return NewStockID(string(u.IngredientID.ID))
}

func (u StockUpdate) CedarEntity() cedar.Entity {
	uid := u.EntityUID()
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
