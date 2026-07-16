package audit

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Writer struct {
	dao *dao.DAO
}

func NewWriter(s *store.Store) *Writer {
	return &Writer{dao: dao.New(s)}
}

func (w *Writer) RecordActivity(ctx *middleware.Context, activity middlewareevents.Activity) error {
	entry := models.AuditEntry{
		ID:          entity.NewAuditEntryID(),
		Action:      activity.Action.String(),
		Resource:    activity.Resource,
		Principal:   activity.Principal,
		StartedAt:   activity.StartedAt,
		CompletedAt: activity.CompletedAt,
		Success:     activity.Success,
		Error:       activity.Error,
		Touches:     activity.Touches,
	}

	return w.dao.Insert(ctx, entry)
}
