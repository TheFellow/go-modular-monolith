package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"

// DrinksLoadedMsg is sent when drinks have been loaded.
type DrinksLoadedMsg struct {
	Drinks          []models.Drink
	IngredientNames map[string]string
	Err             error
}
