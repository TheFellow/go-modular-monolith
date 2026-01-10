package entity

import "github.com/cedar-policy/cedar-go"

const TypeStock = cedar.EntityType("Mixology::Stock")

func StockID(ingredientID cedar.EntityUID) cedar.EntityUID {
	return cedar.NewEntityUID(TypeStock, ingredientID.ID)
}
