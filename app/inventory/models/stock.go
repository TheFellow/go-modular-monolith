package models

import (
	"time"

	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	cedar "github.com/cedar-policy/cedar-go"
)

const StockEntityType = cedar.EntityType("Mixology::Stock")

func NewStockID(ingredientID cedar.EntityUID) cedar.EntityUID {
	return cedar.NewEntityUID(StockEntityType, ingredientID.ID)
}

type Stock struct {
	IngredientID cedar.EntityUID
	Quantity     float64
	Unit         ingredientsmodels.Unit
	LastUpdated  time.Time
}

func (s Stock) EntityUID() cedar.EntityUID {
	return NewStockID(s.IngredientID)
}

func (s Stock) CedarEntity() cedar.Entity {
	uid := s.EntityUID()
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

type AdjustmentReason string

const (
	ReasonReceived  AdjustmentReason = "received"
	ReasonUsed      AdjustmentReason = "used"
	ReasonSpilled   AdjustmentReason = "spilled"
	ReasonExpired   AdjustmentReason = "expired"
	ReasonCorrected AdjustmentReason = "corrected"
)
