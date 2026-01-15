package entity

import "github.com/cedar-policy/cedar-go"

const (
	TypeDrink   = cedar.EntityType("Mixology::Drink")
	PrefixDrink = "drk"
)

func DrinkID(id string) cedar.EntityUID {
	return cedar.NewEntityUID(TypeDrink, cedar.String(id))
}

func NewDrinkID() cedar.EntityUID {
	return NewID(TypeDrink, PrefixDrink)
}
