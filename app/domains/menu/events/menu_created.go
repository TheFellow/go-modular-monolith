package events

import "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"

type MenuCreated struct {
	Menu models.Menu
}
