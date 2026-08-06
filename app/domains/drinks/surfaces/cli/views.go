package cli

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
)

type DrinkRow struct {
	ID          string               `table:"ID" json:"id"`
	Name        string               `table:"NAME" json:"name"`
	Category    string               `table:"CATEGORY" json:"category"`
	Glass       string               `table:"GLASS" json:"glass"`
	Status      string               `table:"STATUS" json:"status"`
	Ingredients int                  `table:"INGREDIENTS" json:"ingredients"`
	Tags        tag.CanonicalStrings `table:"TAGS" json:"tags"`
}

func ToDrinkRow(d *models.Drink) DrinkRow {
	if d == nil {
		return DrinkRow{}
	}
	return DrinkRow{
		ID:          d.ID.String(),
		Name:        d.Name,
		Category:    string(d.Category),
		Glass:       string(d.Glass),
		Status:      string(d.Status),
		Ingredients: len(d.Recipe.Ingredients),
		Tags:        d.Tags.Canonical(),
	}
}

func ToDrinkRows(items []*models.Drink) []DrinkRow {
	rows := make([]DrinkRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, ToDrinkRow(item))
	}
	return rows
}
