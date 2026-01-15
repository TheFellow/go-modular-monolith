package dao

import (
	"context"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

// ListFilter specifies optional filters for listing audit entries.
type ListFilter struct {
	Action        cedar.EntityUID
	Principal     cedar.EntityUID
	Entity        cedar.EntityUID
	StartedAfter  time.Time
	StartedBefore time.Time
	Limit         int
}

func (d *DAO) List(ctx context.Context, filter ListFilter) ([]*models.AuditEntry, error) {
	var out []*models.AuditEntry
	err := d.read(ctx, func(tx *bstore.Tx) error {
		q := bstore.QueryTx[AuditEntryRow](tx)
		if !filter.Action.IsZero() {
			q = q.FilterEqual("Action", filter.Action.String())
		}
		if !filter.Principal.IsZero() {
			q = q.FilterEqual("PrincipalType", string(filter.Principal.Type))
			q = q.FilterEqual("PrincipalID", string(filter.Principal.ID))
		}
		if !filter.StartedAfter.IsZero() {
			q = q.FilterGreaterEqual("StartedAt", filter.StartedAfter)
		}
		if !filter.StartedBefore.IsZero() {
			q = q.FilterLessEqual("StartedAt", filter.StartedBefore)
		}
		q = q.SortDesc("StartedAt")
		if filter.Limit > 0 && filter.Entity.IsZero() {
			q = q.Limit(filter.Limit)
		}

		rows, err := q.List()
		if err != nil {
			return store.MapError(err, "list audit entries")
		}
		entries := make([]*models.AuditEntry, 0, len(rows))
		for _, r := range rows {
			e := toModel(r)
			if !matchesEntityFilter(filter.Entity, e) {
				continue
			}
			entries = append(entries, &e)
		}
		if filter.Limit > 0 && len(entries) > filter.Limit {
			entries = entries[:filter.Limit]
		}
		out = entries
		return nil
	})
	return out, err
}

func matchesEntityFilter(entity cedar.EntityUID, entry models.AuditEntry) bool {
	if entity.IsZero() {
		return true
	}
	if entry.Resource == entity {
		return true
	}
	for _, touched := range entry.Touches {
		if touched == entity {
			return true
		}
	}
	return false
}
