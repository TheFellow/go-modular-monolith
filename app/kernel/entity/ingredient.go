package entity

import cedar "github.com/cedar-policy/cedar-go"

const TypeIngredient = cedar.EntityType("Mixology::Ingredient")

func IngredientID(id string) cedar.EntityUID {
	return cedar.NewEntityUID(TypeIngredient, cedar.String(id))
}
