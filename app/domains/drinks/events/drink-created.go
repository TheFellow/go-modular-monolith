package events

import "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"

type DrinkCreated struct {
	Drink models.Drink
}
