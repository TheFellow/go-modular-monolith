package authz

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/cedar-policy/cedar-go"
)

const MixologyDrinkAction cedar.EntityType = entity.TypeDrink + "::Action"

var (
	ActionList   = cedar.NewEntityUID(MixologyDrinkAction, "list")
	ActionGet    = cedar.NewEntityUID(MixologyDrinkAction, "get")
	ActionCreate = cedar.NewEntityUID(MixologyDrinkAction, "create")
	ActionUpdate = cedar.NewEntityUID(MixologyDrinkAction, "update")
	ActionDelete = cedar.NewEntityUID(MixologyDrinkAction, "delete")
)
