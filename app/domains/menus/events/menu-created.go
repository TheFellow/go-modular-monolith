package events

import "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"

type MenuCreated struct {
	Menu models.Menu
}
