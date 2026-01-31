package events

import "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"

type DrinkRemovedFromMenu struct {
	Menu models.Menu
	Item models.MenuItem
}
