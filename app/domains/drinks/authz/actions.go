package authz

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/cedar-policy/cedar-go"
)

const DrinkAction cedar.EntityType = entity.TypeDrink + "::Action"

var (
	ActionList   = cedar.NewEntityUID(DrinkAction, "list")
	ActionGet    = cedar.NewEntityUID(DrinkAction, "get")
	ActionCreate = cedar.NewEntityUID(DrinkAction, "create")
	ActionUpdate = cedar.NewEntityUID(DrinkAction, "update")
	ActionDelete = cedar.NewEntityUID(DrinkAction, "delete")
)
