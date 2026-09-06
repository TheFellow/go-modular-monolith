package events

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
)

type IngredientDeleted struct {
	Withdraw         bool
	Reason           string
	Ingredient       models.Ingredient
	DeletedAt        time.Time
	Replacement      *models.Ingredient
	ReplacementRatio float64
}
