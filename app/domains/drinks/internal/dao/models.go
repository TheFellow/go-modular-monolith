package dao

import (
	"time"

	"github.com/cedar-policy/cedar-go"
)

type DrinkRow struct {
	ID          string
	Name        string `store:"unique"`
	Category    string `store:"index"`
	Glass       string `store:"index"`
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
