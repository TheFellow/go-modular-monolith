package queries

import (
	"iter"

	auditdao "github.com/TheFellow/go-modular-monolith/app/domains/audit/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) List(ctx store.Context, filter auditdao.ListFilter) iter.Seq2[*models.AuditEntry, error] {
	return q.dao.List(ctx, filter)
}
