package authz

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/cedar-policy/cedar-go"
)

const OrderAction cedar.EntityType = entity.TypeOrder + "::Action"

var (
	ActionList     = cedar.NewEntityUID(OrderAction, "list")
	ActionGet      = cedar.NewEntityUID(OrderAction, "get")
	ActionPlace    = cedar.NewEntityUID(OrderAction, "place")
	ActionComplete = cedar.NewEntityUID(OrderAction, "complete")
	ActionCancel   = cedar.NewEntityUID(OrderAction, "cancel")
)
