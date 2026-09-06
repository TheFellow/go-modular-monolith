package events

import "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"

type MenuDeleted struct{ Menu models.Menu }
