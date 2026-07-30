package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
import "github.com/TheFellow/go-modular-monolith/pkg/paging"

// DrinksLoadedMsg is sent when drinks have been loaded.
type DrinksLoadedMsg struct {
	Drinks []models.Drink
	Err    error
	Next   paging.Cursor
	Token  uint64
}

// DrinkDeletedMsg is sent when a drink has been deleted.
type DrinkDeletedMsg struct {
	Drink *models.Drink
}

// DeleteErrorMsg is sent when delete fails.
type DeleteErrorMsg struct {
	Err error
}
