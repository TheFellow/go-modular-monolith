package cli

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
)

type Menu struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Status      string               `json:"status"`
	CreatedAt   string               `json:"created_at"`
	PublishedAt *string              `json:"published_at,omitempty"`
	Items       []MenuItem           `json:"items,omitempty"`
	Tags        tag.CanonicalStrings `json:"tags"`
}

type MenuItem struct {
	DrinkID      string `json:"drink_id"`
	DisplayName  string `json:"display_name,omitempty"`
	Price        string `json:"price,omitempty"`
	Featured     bool   `json:"featured,omitempty"`
	Availability string `json:"availability"`
	SortOrder    int    `json:"sort_order,omitempty"`
}

func FromDomainMenu(m models.Menu) Menu {
	var publishedAt *string
	if t, ok := m.PublishedAt.Unwrap(); ok {
		s := t.Format("2006-01-02T15:04:05Z07:00")
		publishedAt = &s
	}

	items := make([]MenuItem, 0, len(m.Items))
	for _, item := range m.Items {
		items = append(items, FromDomainMenuItem(item))
	}

	return Menu{
		ID:          m.ID.String(),
		Name:        m.Name,
		Description: m.Description,
		Status:      string(m.Status),
		CreatedAt:   m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		PublishedAt: publishedAt,
		Items:       items,
		Tags:        m.Tags.Canonical(),
	}
}

func FromDomainMenuItem(i models.MenuItem) MenuItem {
	var displayName string
	displayName, _ = i.DisplayName.Unwrap()
	var price string
	if value, ok := i.Price.Unwrap(); ok {
		price = value.String()
	}
	return MenuItem{
		DrinkID:      i.DrinkID.String(),
		DisplayName:  displayName,
		Price:        price,
		Featured:     i.Featured,
		Availability: string(i.Availability),
		SortOrder:    i.SortOrder,
	}
}
