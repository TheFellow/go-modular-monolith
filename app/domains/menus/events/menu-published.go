package events

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
)

type MenuPublished struct {
	Menu models.Menu
}
