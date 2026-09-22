package dao

import (
	"time"

	"github.com/cedar-policy/cedar-go"
)

type DrinkRow struct {
	ID          string
	Revision    uint64 `json:"-" store:"revision"`
	Name        string `store:"unique"`
	Category    string `store:"index=Category+ID"`
	Glass       string `store:"index=Glass+ID"`
	Recipe      RecipeRow
	Description string
	Status      string `store:"index"`
	DeletedAt   *time.Time
}

type RecipeRow struct {
	Ingredients []RecipeIngredientRow
	Steps       []string
	Garnish     string
}

type RecipeIngredientRow struct {
	IngredientID cedar.EntityUID
	Amount       float64
	Unit         string
	Optional     bool
	Substitutes  []cedar.EntityUID
}

func (DrinkRow) StoreModelName() string { return "drinks" }
