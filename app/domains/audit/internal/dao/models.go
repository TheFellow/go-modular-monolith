package dao

import (
	"time"

	cedar "github.com/cedar-policy/cedar-go"
)

type AuditEntryRow struct {
	ID string

	Action string `store:"index"`

	ResourceType string `store:"index"`
	ResourceID   string `store:"index"`

	PrincipalType string `store:"index"`
	PrincipalID   string `store:"index"`

	Touches []cedar.EntityUID

	StartedAt   time.Time `store:"index"`
	CompletedAt time.Time

	Success bool `store:"index"`
	Error   string
}
