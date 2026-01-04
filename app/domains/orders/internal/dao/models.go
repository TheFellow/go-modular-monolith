package dao

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

type OrderRow struct {
	ID          string
	MenuID      string
	Items       []OrderItemRow
	Status      string    `bstore:"index"`
	CreatedAt   time.Time `bstore:"index"`
	CompletedAt optional.Value[time.Time]
	Notes       string
}

type OrderItemRow struct {
	DrinkID  string
	Quantity int
	Notes    string
}
