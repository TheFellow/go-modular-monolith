package entity

import "github.com/cedar-policy/cedar-go"

const (
	TypeOrder   = cedar.EntityType("Mixology::Order")
	PrefixOrder = "ord"
)

func OrderID(id string) cedar.EntityUID {
	return cedar.NewEntityUID(TypeOrder, cedar.String(id))
}

func NewOrderID() cedar.EntityUID {
	return NewID(TypeOrder, PrefixOrder)
}
