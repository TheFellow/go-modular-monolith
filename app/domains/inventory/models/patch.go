package models

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

type Patch struct {
	IngredientID entity.IngredientID
	Reason       AdjustmentReason
	Delta        optional.Value[measurement.Amount]
	CostPerUnit  optional.Value[money.Price]
}

func (p Patch) EntityUID() cedar.EntityUID {
	_ = p.Reason
	_ = p.Delta
	_ = p.CostPerUnit
	return cedar.NewEntityUID(InventoryEntityType, cedar.String(""))
}

func (p Patch) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        p.EntityUID(),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}
