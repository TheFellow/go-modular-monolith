package cli

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
)

type MenuRow struct {
	ID     string `table:"ID" json:"id"`
	Name   string `table:"NAME" json:"name"`
	Status string `table:"STATUS" json:"status"`
	Items  string `table:"ITEMS" json:"items"`
	Desc   string `table:"-" json:"description,omitempty"`
}

type MenuItemRow struct {
	DrinkID      string `table:"DRINK_ID" json:"drink_id"`
	Availability string `table:"AVAILABILITY" json:"availability"`
}

func ToMenuRow(m *models.Menu) MenuRow {
	if m == nil {
		return MenuRow{}
	}
	return MenuRow{
		ID:     m.ID.String(),
		Name:   m.Name,
		Status: string(m.Status),
		Items:  fmt.Sprintf("%d", len(m.Items)),
		Desc:   m.Description,
	}
}

func ToMenuRows(items []*models.Menu) []MenuRow {
	rows := make([]MenuRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, ToMenuRow(item))
	}
	return rows
}

func ToMenuItemRows(items []models.MenuItem) []MenuItemRow {
	rows := make([]MenuItemRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, MenuItemRow{
			DrinkID:      item.DrinkID.String(),
			Availability: string(item.Availability),
		})
	}
	return rows
}
