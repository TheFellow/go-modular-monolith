package dao

import (
	"time"

	domain "github.com/TheFellow/go-modular-monolith/app/ingredients/models"
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

func (i Ingredient) ToDomain() domain.Ingredient {
	return domain.Ingredient{
		ID:          cedar.NewEntityUID(domain.IngredientEntityType, cedar.String(i.ID)),
		Name:        i.Name,
		Category:    domain.Category(i.Category),
		Unit:        domain.Unit(i.Unit),
		Description: i.Description,
	}
}

func FromDomain(i domain.Ingredient) Ingredient {
	return Ingredient{
		ID:          string(i.ID.ID),
		Name:        i.Name,
		Category:    string(i.Category),
		Unit:        string(i.Unit),
		Description: i.Description,
	}
}
