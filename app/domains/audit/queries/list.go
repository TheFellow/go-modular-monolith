package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
)

func (q *Queries) List(ctx context.Context, filter dao.ListFilter) ([]*models.AuditEntry, error) {
	return q.dao.List(ctx, filter)
}
