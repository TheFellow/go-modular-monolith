package models

import (
	menuauthz "github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	cedar "github.com/cedar-policy/cedar-go"
)

type MenuPatch struct {
	MenuID  entity.MenuID
	DrinkID entity.DrinkID
}

func (c MenuPatch) EntityUID() cedar.EntityUID {
	return c.MenuID.EntityUID()
}

func (c MenuPatch) CedarEntity() cedar.Entity {
	return menuauthz.Menu{UID: c.MenuID.EntityUID()}.CedarEntity()
}
