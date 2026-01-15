package entity

import "github.com/cedar-policy/cedar-go"

const (
	TypeMenu   = cedar.EntityType("Mixology::Menu")
	PrefixMenu = "mnu"
)

func MenuID(id string) cedar.EntityUID {
	return cedar.NewEntityUID(TypeMenu, cedar.String(id))
}

func NewMenuID() cedar.EntityUID {
	return NewID(TypeMenu, PrefixMenu)
}
