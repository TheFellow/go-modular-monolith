package dao

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

func toRow(i models.Ingredient) IngredientRow {
	var deletedAt *time.Time
	if t, ok := i.DeletedAt.Unwrap(); ok {
		deletedAt = &t
	}
	return IngredientRow{
		ID:          i.ID.String(),
		Name:        i.Name,
		Category:    string(i.Category),
		Unit:        string(i.Unit),
		Description: i.Description,
		DeletedAt:   deletedAt,
	}
}

func toModel(r IngredientRow) models.Ingredient {
	var deletedAt optional.Value[time.Time]
	if r.DeletedAt != nil {
		deletedAt = optional.Some(*r.DeletedAt)
	} else {
		deletedAt = optional.None[time.Time]()
	}
	return models.Ingredient{
		ID:          entity.IngredientID(cedar.NewEntityUID(entity.TypeIngredient, cedar.String(r.ID))),
		Name:        r.Name,
		Category:    models.Category(r.Category),
		Unit:        models.Unit(r.Unit),
		Description: r.Description,
		DeletedAt:   deletedAt,
	}
}
