package authz

import cedar "github.com/cedar-policy/cedar-go"

var (
	ActionList   = cedar.NewEntityUID(cedar.EntityType("Mixology::Stock::Action"), cedar.String("list"))
	ActionGet    = cedar.NewEntityUID(cedar.EntityType("Mixology::Stock::Action"), cedar.String("get"))
	ActionAdjust = cedar.NewEntityUID(cedar.EntityType("Mixology::Stock::Action"), cedar.String("adjust"))
	ActionSet    = cedar.NewEntityUID(cedar.EntityType("Mixology::Stock::Action"), cedar.String("set"))
)
