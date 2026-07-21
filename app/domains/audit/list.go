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
	Cursor    paging.Cursor
	Limit     int
}

const defaultPageLimit = 100

// List returns a stable cursor page ordered by descending audit-entry ID.
// KSUIDs created within one second are ordered by their random payload rather
// than StartedAt, and separate page requests do not form a database snapshot.
func (m *Module) List(ctx *middleware.Context, req ListRequest) (paging.Page[*models.AuditEntry], error) {
	if req.Limit == 0 {
		req.Limit = defaultPageLimit
	}
	if req.Cursor != "" {
		if _, err := entity.ParseAuditEntryID(string(req.Cursor)); err != nil {
			return paging.Page[*models.AuditEntry]{}, err
		}
	}
	filter := m.listFilter(req)
	return middleware.RunPageQuery(
		m.pipeline,
		ctx,
		authz.ActionList,
		func(ctx store.Context, filter dao.ListFilter, cursor paging.Cursor) iter.Seq2[*models.AuditEntry, error] {
			filter.BeforeID = string(cursor)
			return m.queries.List(ctx, filter)
		},
		func(entry *models.AuditEntry) paging.Cursor { return paging.Cursor(entry.ID.String()) },
		filter,
		paging.Request{Cursor: req.Cursor, Limit: req.Limit},
	)
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
	req.Cursor = ""
	req.Limit = defaultPageLimit
	count := 0
	for {
		page, err := m.List(ctx, req)
		if err != nil {
			return 0, err
		}
		count += len(page.Items)
		if page.Next == "" {
			return count, nil
		}
		req.Cursor = page.Next
	}
}

func (m *Module) GetEntityHistory(ctx *middleware.Context, uid cedar.EntityUID) (paging.Page[*models.AuditEntry], error) {
	return m.List(ctx, ListRequest{Entity: uid})
}

func (m *Module) GetActorActivity(ctx *middleware.Context, principal cedar.EntityUID) (paging.Page[*models.AuditEntry], error) {
	return m.List(ctx, ListRequest{Principal: principal})
}
