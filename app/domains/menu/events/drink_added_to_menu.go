package events

import "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"

type DrinkAddedToMenu struct {
	Menu models.Menu
	Item models.MenuItem
}
