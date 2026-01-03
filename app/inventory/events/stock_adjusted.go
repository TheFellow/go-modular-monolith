package events

import cedar "github.com/cedar-policy/cedar-go"

type StockAdjusted struct {
	IngredientID cedar.EntityUID
	PreviousQty  float64
	NewQty       float64
	Delta        float64
	Reason       string
}
