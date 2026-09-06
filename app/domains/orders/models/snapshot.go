package models

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"time"
)

// AcceptanceSnapshot is immutable evidence of the accepted menu and recipe.
// Plan is the current approved fulfillment; amendments never rewrite acceptance.
type AcceptanceSnapshot struct {
	MenuID   entity.MenuID
	MenuName string
	Items    []ItemSnapshot
}
type ItemSnapshot struct {
	DrinkID     entity.DrinkID
	Name        string
	Quantity    int
	Notes       string
	Price       optional.Value[money.Price]
	Steps       []string
	Garnish     string
	Ingredients []IngredientSelection
}
type IngredientSelection struct {
	OriginalID   entity.IngredientID
	IngredientID entity.IngredientID
	Name         string
	Quantity     float64
	Unit         measurement.Unit
	Optional     bool
	Omitted      bool
	Ratio        float64
}
type Replacement struct {
	OriginalID    entity.IngredientID
	ReplacementID entity.IngredientID
	Ratio         float64
}
type Amendment struct {
	OrderID      entity.OrderID
	Revision     uint64
	Reason       string
	Replacements []Replacement
	Preparation  []PreparationAmendment
}
type AmendmentRecord struct {
	At        time.Time
	Principal string
	Reason    string
	Before    []ItemSnapshot
	After     []ItemSnapshot
}

// PreparationAmendment records approved instructions for a changed line item.
// Nil Steps and absent Garnish preserve the previously approved values.
type PreparationAmendment struct {
	DrinkID entity.DrinkID
	Steps   []string
	Garnish optional.Value[string]
}
