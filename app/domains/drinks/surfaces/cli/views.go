package cli

import "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"

type DrinkRow struct {
	ID   string `table:"ID" json:"id"`
	Name string `table:"NAME" json:"name"`
}

func ToDrinkRow(d *models.Drink) DrinkRow {
	if d == nil {
		return DrinkRow{}
	}
	return DrinkRow{
		ID:   d.ID.String(),
		Name: d.Name,
	}
}

func ToDrinkRows(items []*models.Drink) []DrinkRow {
	rows := make([]DrinkRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, ToDrinkRow(item))
	}
	return rows
}
