package models

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/filter"
)

type ListFilterView struct {
	ID          string    `expr:"id" filter:"Audit entry ID" filter-column:"ID"`
	Action      string    `expr:"action" filter:"Action entity UID" filter-column:"Action"`
	Resource    string    `expr:"resource" filter:"Primary resource entity UID"`
	Principal   string    `expr:"principal" filter:"Principal entity UID"`
	StartedAt   time.Time `expr:"started_at" filter:"Start timestamp" filter-column:"StartedAt"`
	CompletedAt time.Time `expr:"completed_at" filter:"Completion timestamp" filter-column:"CompletedAt"`
	Success     bool      `expr:"success" filter:"Whether the operation succeeded" filter-column:"Success"`
	Error       string    `expr:"error" filter:"Recorded error text" filter-column:"Error"`
}

func ListFilterSchema() filter.Schema[ListFilterView] {
	return filter.NewSchema[ListFilterView](
		`!success && error.contains("permission")`,
		`started_at >= date("2026-07-01T00:00:00Z") && (action == "Mixology::Order::Action::\"place\"" || principal.contains("manager"))`,
	)
}
