package app

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders"
)

func StoreTypes() []any {
	types := make([]any, 0, 8)
	types = append(types, drinks.StoreTypes()...)
	types = append(types, ingredients.StoreTypes()...)
	types = append(types, inventory.StoreTypes()...)
	types = append(types, menu.StoreTypes()...)
	types = append(types, orders.StoreTypes()...)
	return types
}
