package events

import (
	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	cedar "github.com/cedar-policy/cedar-go"
)

type DrinkCreated struct {
	DrinkID     cedar.EntityUID
	Name        string
	Category    models.DrinkCategory
	Glass       models.GlassType
	Recipe      models.Recipe
	Description string
}
