package authz

import cedar "github.com/cedar-policy/cedar-go"

var (
	ActionList        = cedar.NewEntityUID(cedar.EntityType("Mixology::Menu::Action"), cedar.String("list"))
	ActionGet         = cedar.NewEntityUID(cedar.EntityType("Mixology::Menu::Action"), cedar.String("get"))
	ActionCreate      = cedar.NewEntityUID(cedar.EntityType("Mixology::Menu::Action"), cedar.String("create"))
	ActionAddDrink    = cedar.NewEntityUID(cedar.EntityType("Mixology::Menu::Action"), cedar.String("add_drink"))
	ActionRemoveDrink = cedar.NewEntityUID(cedar.EntityType("Mixology::Menu::Action"), cedar.String("remove_drink"))
	ActionPublish     = cedar.NewEntityUID(cedar.EntityType("Mixology::Menu::Action"), cedar.String("publish"))
)
