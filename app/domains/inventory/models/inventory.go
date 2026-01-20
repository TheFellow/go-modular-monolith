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

type Inventory struct {
	ID           entity.InventoryID
	IngredientID entity.IngredientID
	Amount       measurement.Amount
	CostPerUnit  optional.Value[money.Price]
	LastUpdated  time.Time
}

func (s Inventory) EntityUID() cedar.EntityUID {
	uid := s.ID.EntityUID()
	if uid.Type == "" {
		uid = cedar.NewEntityUID(cedar.EntityType(InventoryEntityType), uid.ID)
	}
	return uid
}

func (s Inventory) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:     s.EntityUID(),
		Parents: cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(cedar.RecordMap{
			"IngredientID": s.IngredientID.EntityUID(),
			"Unit":         cedar.String(s.Amount.Unit()),
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
