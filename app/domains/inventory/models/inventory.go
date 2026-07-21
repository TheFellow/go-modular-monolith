package models

import (
	"time"

	inventoryauthz "github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
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
	return cedar.NewEntityUID(InventoryEntityType, s.ID.EntityUID().ID)
}

func (s Inventory) CedarEntity() cedar.Entity {
	return inventoryauthz.Inventory{
		UID: s.EntityUID(), IngredientID: s.IngredientID.EntityUID(), Unit: string(s.Amount.Unit()),
	}.CedarEntity()
}

type AdjustmentReason string

const (
	ReasonReceived  AdjustmentReason = "received"
	ReasonUsed      AdjustmentReason = "used"
	ReasonSpilled   AdjustmentReason = "spilled"
	ReasonExpired   AdjustmentReason = "expired"
	ReasonCorrected AdjustmentReason = "corrected"
)
