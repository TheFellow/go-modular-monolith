package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"

// MenusLoadedMsg is sent when menus have been loaded.
type MenusLoadedMsg struct {
	Menus []models.Menu
	Err   error
}
