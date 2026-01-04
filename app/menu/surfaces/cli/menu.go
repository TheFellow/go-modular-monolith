package cli

import (
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
)

type Menu struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   string     `json:"created_at"`
	PublishedAt *string    `json:"published_at,omitempty"`
	Items       []MenuItem `json:"items,omitempty"`
}

type MenuItem struct {
	DrinkID      string `json:"drink_id"`
	DisplayName  string `json:"display_name,omitempty"`
	Featured     bool   `json:"featured,omitempty"`
	Availability string `json:"availability"`
	SortOrder    int    `json:"sort_order,omitempty"`
}

func FromDomainMenu(m models.Menu) Menu {
	var publishedAt *string
	if m.PublishedAt != nil {
		s := m.PublishedAt.Format("2006-01-02T15:04:05Z07:00")
		publishedAt = &s
	}

	items := make([]MenuItem, 0, len(m.Items))
	for _, item := range m.Items {
		items = append(items, FromDomainMenuItem(item))
	}

	return Menu{
		ID:          string(m.ID.ID),
		Name:        m.Name,
		Description: m.Description,
		Status:      string(m.Status),
		CreatedAt:   m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		PublishedAt: publishedAt,
		Items:       items,
	}
}

func FromDomainMenuItem(i models.MenuItem) MenuItem {
	return MenuItem{
		DrinkID:      string(i.DrinkID.ID),
		DisplayName:  i.DisplayName,
		Featured:     i.Featured,
		Availability: string(i.Availability),
		SortOrder:    i.SortOrder,
	}
}
