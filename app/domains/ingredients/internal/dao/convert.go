package dao

import "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"

func toRow(i models.Ingredient) IngredientRow {
	return IngredientRow{
		ID:          string(i.ID.ID),
		Name:        i.Name,
		Category:    string(i.Category),
		Unit:        string(i.Unit),
		Description: i.Description,
	}
}

func toModel(r IngredientRow) models.Ingredient {
	return models.Ingredient{
		ID:          models.NewIngredientID(r.ID),
		Name:        r.Name,
		Category:    models.Category(r.Category),
		Unit:        models.Unit(r.Unit),
		Description: r.Description,
	}
}
