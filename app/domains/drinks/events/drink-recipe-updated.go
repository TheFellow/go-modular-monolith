package events

import "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"

type DrinkRecipeUpdated struct {
	Previous models.Drink
	Current  models.Drink
}
