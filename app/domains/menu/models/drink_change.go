package models

import cedar "github.com/cedar-policy/cedar-go"

type MenuDrinkChange struct {
	MenuID  cedar.EntityUID
	DrinkID cedar.EntityUID
}

func (c MenuDrinkChange) EntityUID() cedar.EntityUID {
	return c.MenuID
}

func (c MenuDrinkChange) CedarEntity() cedar.Entity {
	uid := c.MenuID
	if string(uid.ID) == "" {
		uid = cedar.NewEntityUID(MenuEntityType, cedar.String(""))
	}
	return cedar.Entity{
		UID:        uid,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}
