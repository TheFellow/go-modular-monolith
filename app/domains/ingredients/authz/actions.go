package authz

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/cedar-policy/cedar-go"
)

const IngredientAction cedar.EntityType = entity.TypeIngredient + "::Action"

var (
	ActionList   = cedar.NewEntityUID(IngredientAction, "list")
	ActionGet    = cedar.NewEntityUID(IngredientAction, "get")
	ActionCreate = cedar.NewEntityUID(IngredientAction, "create")
	ActionUpdate = cedar.NewEntityUID(IngredientAction, "update")
)
