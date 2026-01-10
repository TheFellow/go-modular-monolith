package events

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
)

type DrinkDeleted struct {
	Drink     models.Drink
	DeletedAt time.Time
}
