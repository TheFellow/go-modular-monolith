package models

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/filter"
)

type ListFilterView struct {
	ID        string    `expr:"id" filter:"Order ID" filter-column:"ID"`
	MenuID    string    `expr:"menu_id" filter:"Menu ID" filter-column:"MenuID"`
	Status    string    `expr:"status" filter:"Order status" filter-column:"Status"`
	CreatedAt time.Time `expr:"created_at" filter:"Creation timestamp" filter-column:"CreatedAt"`
	Notes     string    `expr:"notes" filter:"Order notes" filter-column:"Notes"`
}

func ListFilterSchema() filter.Schema[ListFilterView] {
	return filter.NewSchema[ListFilterView](
		`status in ["pending", "completed"] && !notes.contains("test")`,
		`menu_id == "menu_123" || created_at >= date("2026-07-01T00:00:00Z")`,
	)
}
