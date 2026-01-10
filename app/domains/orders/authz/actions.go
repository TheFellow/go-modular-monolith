package authz

import "github.com/cedar-policy/cedar-go"

var (
	ActionList     = cedar.NewEntityUID(cedar.EntityType("Mixology::Order::Action"), cedar.String("list"))
	ActionGet      = cedar.NewEntityUID(cedar.EntityType("Mixology::Order::Action"), cedar.String("get"))
	ActionPlace    = cedar.NewEntityUID(cedar.EntityType("Mixology::Order::Action"), cedar.String("place"))
	ActionComplete = cedar.NewEntityUID(cedar.EntityType("Mixology::Order::Action"), cedar.String("complete"))
	ActionCancel   = cedar.NewEntityUID(cedar.EntityType("Mixology::Order::Action"), cedar.String("cancel"))
)
