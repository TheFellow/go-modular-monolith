package authz

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/cedar-policy/cedar-go"
)

const MenuAction cedar.EntityType = entity.TypeMenu + "::Action"

var (
	ActionList        = cedar.NewEntityUID(MenuAction, "list")
	ActionGet         = cedar.NewEntityUID(MenuAction, "get")
	ActionCreate      = cedar.NewEntityUID(MenuAction, "create")
	ActionUpdate      = cedar.NewEntityUID(MenuAction, "update")
	ActionDelete      = cedar.NewEntityUID(MenuAction, "delete")
	ActionAddDrink    = cedar.NewEntityUID(MenuAction, "add_drink")
	ActionRemoveDrink = cedar.NewEntityUID(MenuAction, "remove_drink")
	ActionPublish     = cedar.NewEntityUID(MenuAction, "publish")
	ActionDraft       = cedar.NewEntityUID(MenuAction, "draft")
)
