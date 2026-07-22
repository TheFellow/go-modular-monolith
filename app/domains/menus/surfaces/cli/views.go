package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

type MenuRow struct {
	ID          string `table:"ID" json:"id,omitempty"`
	Name        string `table:"NAME" json:"name"`
	Status      string `table:"STATUS" json:"status,omitempty"`
	Items       string `table:"ITEMS" json:"items,omitempty"`
	CreatedAt   string `table:"CREATED_AT" json:"created_at,omitempty"`
	PublishedAt string `table:"PUBLISHED_AT" json:"published_at,omitempty"`
	Desc        string `table:"-" json:"description,omitempty"`
}

type MenuItemRow struct {
	DrinkID      string `table:"DRINK_ID" json:"drink_id"`
	DisplayName  string `table:"DISPLAY_NAME" json:"display_name,omitempty"`
	Price        string `table:"PRICE" json:"price,omitempty"`
	Featured     bool   `table:"FEATURED" json:"featured"`
	Availability string `table:"AVAILABILITY" json:"availability"`
	SortOrder    int    `table:"SORT_ORDER" json:"sort_order"`
}

func ToMenuRow(m *models.Menu) MenuRow {
	if m == nil {
		return MenuRow{}
	}
	var publishedAt string
	if t, ok := m.PublishedAt.Unwrap(); ok {
		publishedAt = formatTime(t)
	}
	return MenuRow{
		ID:          m.ID.String(),
		Name:        m.Name,
		Status:      string(m.Status),
		Items:       fmt.Sprintf("%d", len(m.Items)),
		CreatedAt:   formatTime(m.CreatedAt),
		PublishedAt: publishedAt,
		Desc:        m.Description,
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
		displayName, _ := item.DisplayName.Unwrap()
		var price string
		if p, ok := item.Price.Unwrap(); ok {
			price = p.String()
		}
		rows = append(rows, MenuItemRow{
			DrinkID:      item.DrinkID.String(),
			DisplayName:  displayName,
			Price:        price,
			Featured:     item.Featured,
			Availability: string(item.Availability),
			SortOrder:    item.SortOrder,
		})
	}
	return rows
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
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
