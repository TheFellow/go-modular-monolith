package events

import "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"

type DrinkUpdated struct {
	Drink models.Drink
}
