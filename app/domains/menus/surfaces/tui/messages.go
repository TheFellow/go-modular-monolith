package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"

// MenusLoadedMsg is sent when menus have been loaded.
type MenusLoadedMsg struct {
	Menus []models.Menu
	Err   error
}

// MenuDeletedMsg is sent when a menu has been deleted.
type MenuDeletedMsg struct {
	Menu *models.Menu
}

// MenuPublishedMsg is sent when a menu has been published.
type MenuPublishedMsg struct {
	Menu *models.Menu
}

// DeleteErrorMsg is sent when a delete operation fails.
type DeleteErrorMsg struct {
	Err error
}

// PublishErrorMsg is sent when a publish operation fails.
type PublishErrorMsg struct {
	Err error
}
