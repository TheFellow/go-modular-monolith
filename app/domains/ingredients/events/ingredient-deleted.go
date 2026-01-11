package events

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
)

type IngredientDeleted struct {
	Ingredient models.Ingredient
	DeletedAt  time.Time
}
