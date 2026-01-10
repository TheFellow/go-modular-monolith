package events

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
)

type MenuPublished struct {
	Menu models.Menu
}
