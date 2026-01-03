package events

import cedar "github.com/cedar-policy/cedar-go"

type IngredientRestocked struct {
	IngredientID cedar.EntityUID
	NewQty       float64
}
