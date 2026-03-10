package dao

import (
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

func (d *DAO) List(ctx store.Context, filter ListFilter) ([]*models.AuditEntry, error) {
	var out []*models.AuditEntry
	err := store.Read(ctx, func(tx *bstore.Tx) error {
		q := d.query(tx, filter)
		rows, err := q.List()
		if err != nil {
			return store.MapError(err, "list audit entries")
		}
		entries := make([]*models.AuditEntry, 0, len(rows))
		for _, r := range rows {
			e := toModel(r)
			entries = append(entries, &e)
		}
		if filter.Limit > 0 && !filter.Entity.IsZero() && len(entries) > filter.Limit {
			entries = entries[:filter.Limit]
		}
		out = entries
		return nil
	})
	return out, err
}

func (d *DAO) Count(ctx store.Context, filter ListFilter) (int, error) {
	var count int
	err := store.Read(ctx, func(tx *bstore.Tx) error {
		q := d.query(tx, filter)

		var err error
		count, err = q.Count()
		if err != nil {
			return store.MapError(err, "count audit entries")
		}
		if filter.Limit > 0 && !filter.Entity.IsZero() && count > filter.Limit {
			count = filter.Limit
		}
		return nil
	})
	return count, err
}

func (d *DAO) query(tx *bstore.Tx, filter ListFilter) *bstore.Query[AuditEntryRow] {
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
	if !filter.Entity.IsZero() {
		q = q.FilterFn(func(r AuditEntryRow) bool {
			return matchesEntityFilterRow(filter.Entity, r)
		})
	}
	q = q.SortDesc("StartedAt")
	if filter.Limit > 0 && filter.Entity.IsZero() {
		q = q.Limit(filter.Limit)
	}
	return q
}

func matchesEntityFilterRow(entity cedar.EntityUID, row AuditEntryRow) bool {
	if entity.IsZero() {
		return true
	}
	if row.ResourceType == string(entity.Type) && row.ResourceID == string(entity.ID) {
		return true
	}
	for _, touched := range row.Touches {
		if touched == entity {
			return true
		}
	}
	return false
}
