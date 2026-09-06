package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	cedar "github.com/cedar-policy/cedar-go"
)

func toRow(i models.Ingredient) IngredientRow {

	return IngredientRow{
		ID:          i.ID.String(),
		Revision:    i.Revision,
		Name:        i.Name,
		Category:    string(i.Category),
		Unit:        string(i.Unit),
		Description: i.Description,
	}
}

func toModel(r IngredientRow) models.Ingredient {
	return models.Ingredient{
		ID:          entity.IngredientID(cedar.NewEntityUID(entity.TypeIngredient, cedar.String(r.ID))),
		Revision:    r.Revision,
		Name:        r.Name,
		Category:    models.Category(r.Category),
		Unit:        measurement.Unit(r.Unit),
		Description: r.Description,
	}
}
