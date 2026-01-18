package models

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	cedar "github.com/cedar-policy/cedar-go"
)

type MenuDrinkChange struct {
	MenuID  entity.MenuID
	DrinkID entity.DrinkID
}

func (c MenuDrinkChange) EntityUID() cedar.EntityUID {
	return c.MenuID.EntityUID()
}

func (c MenuDrinkChange) CedarEntity() cedar.Entity {
	return cedar.Entity{
		UID:        c.MenuID.EntityUID(),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}
