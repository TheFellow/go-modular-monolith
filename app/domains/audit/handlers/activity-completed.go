package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
)

type ActivityCompleted struct {
	dao *dao.DAO
}

func NewActivityCompleted() *ActivityCompleted {
	return &ActivityCompleted{dao: dao.New()}
}

func (h *ActivityCompleted) Handle(ctx *middleware.Context, e middlewareevents.ActivityCompleted) error {
	entry := models.AuditEntry{
		ID:          entity.NewAuditEntryID(),
		Action:      e.Activity.Action.String(),
		Resource:    e.Activity.Resource,
		Principal:   e.Activity.Principal,
		StartedAt:   e.Activity.StartedAt,
		CompletedAt: e.Activity.CompletedAt,
		Success:     e.Activity.Success,
		Error:       e.Activity.Error,
		Touches:     e.Activity.Touches,
	}

	return h.dao.Insert(ctx, entry)
}
