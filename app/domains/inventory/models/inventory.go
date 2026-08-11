package models

import (
	"time"

	inventoryauthz "github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

const InventoryEntityType = entity.TypeInventory

type Inventory struct {
	ID           entity.InventoryID
	Revision     uint64 `json:"revision"`
	IngredientID entity.IngredientID
	Amount       measurement.Amount
	Reserved     measurement.Amount
	CostPerUnit  optional.Value[money.Price]
	LastUpdated  time.Time
	Tags         tag.Tags
}

func (s Inventory) Available() measurement.Amount {
	if s.Amount == nil {
		return nil
	}
	if s.Reserved == nil {
		return s.Amount
	}
	reserved, err := s.Reserved.Convert(s.Amount.Unit())
	if err != nil {
		return s.Amount
	}
	available, err := s.Amount.Sub(reserved)
	if err != nil || available.Value() < 0 {
		return measurement.MustAmount(0, s.Amount.Unit())
	}
	return available
}

func (s Inventory) ReservedAmount() measurement.Amount {
	if s.Reserved != nil {
		return s.Reserved
	}
	if s.Amount != nil {
		return measurement.MustAmount(0, s.Amount.Unit())
	}
	return nil
}

func (s Inventory) EntityUID() cedar.EntityUID {
	return cedar.NewEntityUID(InventoryEntityType, s.ID.EntityUID().ID)
}

func (s *Inventory) SetTags(tags tag.Tags) { s.Tags = tags }

func (s Inventory) CedarEntity() cedar.Entity {
	return inventoryauthz.Inventory{
		UID: s.EntityUID(), IngredientID: s.IngredientID.EntityUID(), Unit: string(s.Amount.Unit()), Tags: s.Tags.Map(),
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
