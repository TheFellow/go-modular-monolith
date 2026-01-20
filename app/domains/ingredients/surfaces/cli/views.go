package cli

import "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"

type IngredientRow struct {
	ID       string `table:"ID" json:"id"`
	Name     string `table:"NAME" json:"name"`
	Category string `table:"CATEGORY" json:"category"`
	Unit     string `table:"UNIT" json:"unit"`
	Desc     string `table:"-" json:"description,omitempty"`
}

func ToIngredientRow(i *models.Ingredient) IngredientRow {
	if i == nil {
		return IngredientRow{}
	}
	return IngredientRow{
		ID:       i.ID.String(),
		Name:     i.Name,
		Category: string(i.Category),
		Unit:     string(i.Unit),
		Desc:     i.Description,
	}
}

func ToIngredientRows(items []*models.Ingredient) []IngredientRow {
	rows := make([]IngredientRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, ToIngredientRow(item))
	}
	return rows
}
