package dao

import (
	"time"

	"github.com/cedar-policy/cedar-go"
)

type DrinkRow struct {
	ID          string
	Name        string `bstore:"unique"`
	Category    string `bstore:"index"`
	Glass       string `bstore:"index"`
	Recipe      RecipeRow
	Description string
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
