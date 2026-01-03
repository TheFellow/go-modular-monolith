package events

import cedar "github.com/cedar-policy/cedar-go"

type IngredientDepleted struct {
	IngredientID cedar.EntityUID
}
