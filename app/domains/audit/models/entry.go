package models

import (
	"time"

	auditauthz "github.com/TheFellow/go-modular-monolith/app/domains/audit/authz"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	cedar "github.com/cedar-policy/cedar-go"
)

const AuditEntryEntityType = entity.TypeAuditEntry

type AuditEntry struct {
	ID entity.AuditEntryID

	Action    string
	Resource  cedar.EntityUID
	Principal cedar.EntityUID

	StartedAt   time.Time
	CompletedAt time.Time

	Success bool
	Error   string

	Touches []cedar.EntityUID
}

func (e AuditEntry) CedarEntity() cedar.Entity {
	return auditauthz.AuditEntry{UID: e.ID.EntityUID()}.CedarEntity()
}
