package audit

import (
	"iter"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

type ListRequest struct {
	Action    cedar.EntityUID
	Principal cedar.EntityUID
	Entity    cedar.EntityUID
	From      time.Time
	To        time.Time
	Limit     int
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) ([]*models.AuditEntry, error) {
	if req.Limit > 0 {
		page, err := m.ListPage(ctx, req, paging.Request{Limit: req.Limit})
		return page.Items, err
	}
	return middleware.RunListQuery(m.pipeline, ctx, authz.ActionList, m.list, req)
}

// ListPage returns a stable cursor page ordered by descending audit-entry ID.
func (m *Module) ListPage(ctx *middleware.Context, req ListRequest, pageRequest paging.Request) (paging.Page[*models.AuditEntry], error) {
	_, err := entity.ParseAuditEntryID(string(pageRequest.Cursor))
	if err != nil {
		return paging.Page[*models.AuditEntry]{}, err
	}
	filter := m.listFilter(req)
	return middleware.RunPageQuery(
		m.pipeline,
		ctx,
		authz.ActionList,
		func(ctx store.Context, filter dao.ListFilter, cursor paging.Cursor) iter.Seq2[*models.AuditEntry, error] {
			filter.BeforeID = string(cursor)
			return m.queries.All(ctx, filter)
		},
		func(entry *models.AuditEntry) paging.Cursor { return paging.Cursor(entry.ID.String()) },
		filter,
		pageRequest,
	)
}

func (m *Module) list(ctx store.Context, req ListRequest) ([]*models.AuditEntry, error) {
	return m.queries.List(ctx, m.listFilter(req))
}

func (m *Module) listFilter(req ListRequest) dao.ListFilter {
	return dao.ListFilter{
		Action:        req.Action,
		Principal:     req.Principal,
		Entity:        req.Entity,
		StartedAfter:  req.From,
		StartedBefore: req.To,
	}
}

func (m *Module) Count(ctx *middleware.Context, req ListRequest) (int, error) {
	entries, err := m.List(ctx, req)
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}

func (m *Module) GetEntityHistory(ctx *middleware.Context, uid cedar.EntityUID) ([]*models.AuditEntry, error) {
	return m.List(ctx, ListRequest{Entity: uid})
}

func (m *Module) GetActorActivity(ctx *middleware.Context, principal cedar.EntityUID) ([]*models.AuditEntry, error) {
	return m.List(ctx, ListRequest{Principal: principal})
}
