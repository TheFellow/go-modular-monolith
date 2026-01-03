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

type AdjustmentReason string

const (
	ReasonReceived  AdjustmentReason = "received"
	ReasonUsed      AdjustmentReason = "used"
	ReasonSpilled   AdjustmentReason = "spilled"
	ReasonExpired   AdjustmentReason = "expired"
	ReasonCorrected AdjustmentReason = "corrected"
)
