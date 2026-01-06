package entity

import cedar "github.com/cedar-policy/cedar-go"

const TypeDrink = cedar.EntityType("Mixology::Drink")

func DrinkID(id string) cedar.EntityUID {
	return cedar.NewEntityUID(TypeDrink, cedar.String(id))
}
