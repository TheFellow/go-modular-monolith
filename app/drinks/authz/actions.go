package authz

import cedar "github.com/cedar-policy/cedar-go"

var (
	ActionList   = cedar.NewEntityUID(cedar.EntityType("Mixology::Drinks::Action"), cedar.String("list"))
	ActionGet    = cedar.NewEntityUID(cedar.EntityType("Mixology::Drinks::Action"), cedar.String("get"))
	ActionCreate = cedar.NewEntityUID(cedar.EntityType("Mixology::Drinks::Action"), cedar.String("create"))
)
