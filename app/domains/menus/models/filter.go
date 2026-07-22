package models

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/filter"
)

type ListFilterView struct {
	ID          string    `expr:"id" filter:"Menu ID" filter-column:"ID"`
	Name        string    `expr:"name" filter:"Menu name" filter-column:"Name"`
	Description string    `expr:"description" filter:"Menu description" filter-column:"Description"`
	Status      string    `expr:"status" filter:"Menu status" filter-column:"Status"`
	CreatedAt   time.Time `expr:"created_at" filter:"Creation timestamp" filter-column:"CreatedAt"`
}

func ListFilterSchema() filter.Schema[ListFilterView] {
	return filter.NewSchema[ListFilterView](
		`status == "published" && name.contains("summer")`,
		`status == "draft" || created_at >= date("2026-07-01T00:00:00Z")`,
	)
}
