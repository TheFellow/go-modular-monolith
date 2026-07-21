package audit

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
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
	unbounded := req
	unbounded.Limit = 0
	entries, err := middleware.RunListQuery(m.pipeline, ctx, authz.ActionList, m.list, unbounded)
	if err != nil {
		return nil, err
	}
	if req.Limit > 0 && len(entries) > req.Limit {
		entries = entries[:req.Limit]
	}
	return entries, nil
}

func (m *Module) list(ctx store.Context, req ListRequest) ([]*models.AuditEntry, error) {
	filter := dao.ListFilter{
		Action:        req.Action,
		Principal:     req.Principal,
		Entity:        req.Entity,
		StartedAfter:  req.From,
		StartedBefore: req.To,
		Limit:         req.Limit,
	}
	return m.queries.List(ctx, filter)
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
