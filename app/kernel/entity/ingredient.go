package entity

import "github.com/cedar-policy/cedar-go"

const (
	TypeIngredient   = cedar.EntityType("Mixology::Ingredient")
	PrefixIngredient = "ing"
)

func IngredientID(id string) cedar.EntityUID {
	return cedar.NewEntityUID(TypeIngredient, cedar.String(id))
}

func NewIngredientID() cedar.EntityUID {
	return NewID(TypeIngredient, PrefixIngredient)
}
