package models

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	cedar "github.com/cedar-policy/cedar-go"
)

type Update struct {
	IngredientID entity.IngredientID
	Quantity     float64
	CostPerUnit  money.Price
}

func (u Update) EntityUID() cedar.EntityUID {
	return NewInventoryID(u.IngredientID).EntityUID()
}

func (u Update) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        u.EntityUID(),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}
