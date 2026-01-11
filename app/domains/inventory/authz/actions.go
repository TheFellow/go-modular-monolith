package authz

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/cedar-policy/cedar-go"
)

const InventoryAction cedar.EntityType = entity.TypeInventory + "::Action"

var (
	ActionList   = cedar.NewEntityUID(InventoryAction, "list")
	ActionGet    = cedar.NewEntityUID(InventoryAction, "get")
	ActionAdjust = cedar.NewEntityUID(InventoryAction, "adjust")
	ActionSet    = cedar.NewEntityUID(InventoryAction, "set")
)
