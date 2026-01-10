package authz

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/cedar-policy/cedar-go"
)

const StockAction cedar.EntityType = entity.TypeStock + "::Action"

var (
	ActionList   = cedar.NewEntityUID(StockAction, "list")
	ActionGet    = cedar.NewEntityUID(StockAction, "get")
	ActionAdjust = cedar.NewEntityUID(StockAction, "adjust")
	ActionSet    = cedar.NewEntityUID(StockAction, "set")
)
