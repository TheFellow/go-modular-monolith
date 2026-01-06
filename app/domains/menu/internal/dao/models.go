package dao

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

type MenuRow struct {
	ID          string
	Name        string `bstore:"unique"`
	Description string
	Items       []MenuItemRow
	Status      string    `bstore:"index"`
	CreatedAt   time.Time `bstore:"index"`
	PublishedAt optional.Value[time.Time]
}

type MenuItemRow struct {
	DrinkID      string
	DisplayName  optional.Value[string]
	Price        optional.Value[money.Price]
	Featured     bool
	Availability string
	SortOrder    int
}
