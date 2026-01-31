package queries

import (
	auditdao "github.com/TheFellow/go-modular-monolith/app/domains/audit/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) List(ctx store.Context, filter auditdao.ListFilter) ([]*models.AuditEntry, error) {
	return q.dao.List(ctx, filter)
}

func (q *Queries) Count(ctx store.Context, filter auditdao.ListFilter) (int, error) {
	return q.dao.Count(ctx, filter)
}
