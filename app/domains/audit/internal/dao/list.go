package dao

import (
	"iter"
	"slices"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	appfilter "github.com/TheFellow/go-modular-monolith/pkg/filter"
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
	BeforeID      string
	Expression    *appfilter.Expression[models.ListFilterView]
}

// List returns an ordered sequence that remains inside its bstore read
// transaction for the duration of iteration.
func (d *DAO) List(ctx store.Context, filter ListFilter) iter.Seq2[*models.AuditEntry, error] {
	return func(yield func(*models.AuditEntry, error) bool) {
		err := d.store.ReadContext(ctx, func(tx *bstore.Tx) error {
			for row, err := range d.query(tx, filter).All() {
				if err != nil {
					return store.MapError(err, "iterate audit entries")
				}
				entry := toModel(row)
				if !yield(&entry, nil) {
					return nil
				}
			}
			return nil
		})
		if err != nil {
			yield(nil, err)
		}
	}
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
	if filter.BeforeID != "" {
		q = q.FilterLess("ID", filter.BeforeID)
	}
	if !filter.Entity.IsZero() {
		q = q.FilterFn(func(r AuditEntryRow) bool {
			return matchesEntityFilterRow(filter.Entity, r)
		})
	}
	q = appfilter.ApplyBstore(q, filter.Expression, func(r AuditEntryRow) models.ListFilterView {
		return models.ListFilterView{
			ID: r.ID, Action: r.Action,
			Resource:  cedar.EntityUID{Type: cedar.EntityType(r.ResourceType), ID: cedar.String(r.ResourceID)}.String(),
			Principal: cedar.EntityUID{Type: cedar.EntityType(r.PrincipalType), ID: cedar.String(r.PrincipalID)}.String(),
			StartedAt: r.StartedAt, CompletedAt: r.CompletedAt, Success: r.Success, Error: r.Error,
		}
	})
	q = q.SortDesc("ID")
	return q
}

func matchesEntityFilterRow(entity cedar.EntityUID, row AuditEntryRow) bool {
	if entity.IsZero() {
		return true
	}
	if row.ResourceType == string(entity.Type) && row.ResourceID == string(entity.ID) {
		return true
	}
	return slices.Contains(row.Touches, entity)
}
