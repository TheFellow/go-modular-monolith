package models

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

const InventoryEntityType = entity.TypeInventory

func NewInventoryID(ingredientID cedar.EntityUID) cedar.EntityUID {
	return entity.InventoryID(ingredientID)
}

type Inventory struct {
	IngredientID cedar.EntityUID
	Quantity     float64
	Unit         measurement.Unit
	CostPerUnit  optional.Value[money.Price]
	LastUpdated  time.Time
}

func (s Inventory) EntityUID() cedar.EntityUID {
	return NewInventoryID(s.IngredientID)
}

func (s Inventory) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:     s.EntityUID(),
		Parents: cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(cedar.RecordMap{
			"IngredientID": s.IngredientID,
			"Unit":         cedar.String(s.Unit),
		}),
		Tags: cedar.NewRecord(nil),
	}
}

type AdjustmentReason string

const (
	ReasonReceived  AdjustmentReason = "received"
	ReasonUsed      AdjustmentReason = "used"
	ReasonSpilled   AdjustmentReason = "spilled"
	ReasonExpired   AdjustmentReason = "expired"
	ReasonCorrected AdjustmentReason = "corrected"
)
