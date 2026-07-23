package dao

import (
	"time"

	cedar "github.com/cedar-policy/cedar-go"
)

type OrderRow struct {
	ID          string
	MenuID      string `bstore:"index"`
	Items       []OrderItemRow
	Status      string    `bstore:"index"`
	CreatedAt   time.Time `bstore:"index"`
	CompletedAt *time.Time
	Notes       string
	DeletedAt   *time.Time
}

type OrderItemRow struct {
	DrinkID  cedar.EntityUID
	Quantity int
	Notes    string
}
