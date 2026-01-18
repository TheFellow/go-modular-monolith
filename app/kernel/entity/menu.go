package entity

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

const (
	TypeMenu   = cedar.EntityType("Mixology::Menu")
	PrefixMenu = "mnu"
)

// MenuID is a strongly typed ID for menu entities.
type MenuID cedar.EntityUID

// NewMenuID generates a new menu ID with a KSUID.
func NewMenuID() MenuID {
	return MenuID(NewID(TypeMenu, PrefixMenu))
}

// ParseMenuID creates a menu ID from a stored string.
func ParseMenuID(id string) (MenuID, error) {
	if id == "" {
		return MenuID(cedar.NewEntityUID(TypeMenu, cedar.String(""))), nil
	}
	if !strings.HasPrefix(id, PrefixMenu+"-") {
		return MenuID{}, errors.Invalidf("invalid menu id prefix: %s", id)
	}
	return MenuID(cedar.NewEntityUID(TypeMenu, cedar.String(id))), nil
}

// EntityUID converts the ID to a cedar.EntityUID.
func (id MenuID) EntityUID() cedar.EntityUID {
	return cedar.EntityUID(id)
}

// String returns the string form of the ID.
func (id MenuID) String() string {
	return string(cedar.EntityUID(id).ID)
}

// IsZero returns true when the ID is unset.
func (id MenuID) IsZero() bool {
	return cedar.EntityUID(id).ID == ""
}
