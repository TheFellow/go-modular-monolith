package authz

import cedar "github.com/cedar-policy/cedar-go"

var (
	ActionList   = cedar.NewEntityUID(cedar.EntityType("Mixology::Drink::Action"), cedar.String("list"))
	ActionGet    = cedar.NewEntityUID(cedar.EntityType("Mixology::Drink::Action"), cedar.String("get"))
	ActionCreate = cedar.NewEntityUID(cedar.EntityType("Mixology::Drink::Action"), cedar.String("create"))
	ActionUpdate = cedar.NewEntityUID(cedar.EntityType("Mixology::Drink::Action"), cedar.String("update"))
)
