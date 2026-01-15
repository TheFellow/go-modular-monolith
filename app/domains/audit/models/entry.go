package models

import (
	"time"

	cedar "github.com/cedar-policy/cedar-go"
)

const AuditEntryEntityType = cedar.EntityType("Mixology::AuditEntry")

type AuditEntry struct {
	ID cedar.EntityUID

	Action    string
	Resource  cedar.EntityUID
	Principal cedar.EntityUID

	StartedAt   time.Time
	CompletedAt time.Time

	Success bool
	Error   string

	Touches []cedar.EntityUID
}

