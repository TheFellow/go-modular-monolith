package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"time"

	cedar "github.com/cedar-policy/cedar-go"
)

type OrderRow struct {
	ID                 string
	Acceptance         models.AcceptanceSnapshot
	Plan               []models.ItemSnapshot
	Amendments         []models.AmendmentRecord
	CancelledAt        *time.Time
	CancellationReason string
	Revision           uint64 `json:"-" store:"revision"`
	MenuID             string `store:"index"`
	Items              []OrderItemRow
	IngredientUsage    []IngredientUsageRow
	BlockedIngredients []string
	Status             string    `store:"index"`
	CreatedAt          time.Time `store:"index"`
	CompletedAt        *time.Time
	Notes              string
	DeletedAt          *time.Time
}

type IngredientUsageRow struct {
	IngredientID string
	Name         string
	Quantity     float64
	Unit         string
}

type OrderItemRow struct {
	DrinkID  cedar.EntityUID
	Quantity int
	Notes    string
}
