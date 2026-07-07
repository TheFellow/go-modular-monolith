package audit

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
)

func (m *Module) RecordActivity(ctx *middleware.Context, activity middlewareevents.Activity) error {
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

	return m.dao.Insert(ctx, entry)
}
