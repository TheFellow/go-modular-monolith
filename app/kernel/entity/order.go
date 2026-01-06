package entity

import cedar "github.com/cedar-policy/cedar-go"

const TypeOrder = cedar.EntityType("Mixology::Order")

func OrderID(id string) cedar.EntityUID {
	return cedar.NewEntityUID(TypeOrder, cedar.String(id))
}
