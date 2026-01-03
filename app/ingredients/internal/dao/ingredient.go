package dao

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	cedar "github.com/cedar-policy/cedar-go"
)

type Ingredient struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Category    string     `json:"category"`
	Unit        string     `json:"unit"`
	Description string     `json:"description,omitempty"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

func (i Ingredient) ToDomain() models.Ingredient {
	return models.Ingredient{
		ID:          cedar.NewEntityUID(models.IngredientEntityType, cedar.String(i.ID)),
		Name:        i.Name,
		Category:    models.Category(i.Category),
		Unit:        models.Unit(i.Unit),
		Description: i.Description,
	}
}

func FromDomain(i models.Ingredient) Ingredient {
	return Ingredient{
		ID:          string(i.ID.ID),
		Name:        i.Name,
		Category:    string(i.Category),
		Unit:        string(i.Unit),
		Description: i.Description,
	}
}

func (i Ingredient) EntityUID() cedar.EntityUID {
	return models.NewIngredientID(i.ID)
}
