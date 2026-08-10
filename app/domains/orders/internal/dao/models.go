package dao

import (
	"time"

	cedar "github.com/cedar-policy/cedar-go"
)

type OrderRow struct {
	ID                 string
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
