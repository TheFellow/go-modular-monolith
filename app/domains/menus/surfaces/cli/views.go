package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type MenuRow struct {
	ID     string `table:"ID" json:"id,omitempty"`
	Name   string `table:"NAME" json:"name"`
	Status string `table:"STATUS" json:"status,omitempty"`
	Items  string `table:"ITEMS" json:"items,omitempty"`
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

func TemplateCreate() MenuRow {
	return MenuRow{
		Name: "Summer Cocktails",
		Desc: "Refreshing drinks for warm weather",
	}
}

func DecodeCreate(r io.Reader) (*models.Menu, error) {
	var row MenuRow
	if err := json.NewDecoder(r).Decode(&row); err != nil {
		return nil, errors.Invalidf("parse menu json: %w", err)
	}
	return &models.Menu{
		Name:        row.Name,
		Description: row.Desc,
	}, nil
}
