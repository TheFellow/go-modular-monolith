package dao

import (
	"time"

	cedar "github.com/cedar-policy/cedar-go"
)

type AuditEntryRow struct {
	ID string

	Action string `bstore:"index"`

	ResourceType string `bstore:"index"`
	ResourceID   string `bstore:"index"`

	PrincipalType string `bstore:"index"`
	PrincipalID   string `bstore:"index"`

	Touches []cedar.EntityUID

	StartedAt   time.Time `bstore:"index"`
	CompletedAt time.Time

	Success bool `bstore:"index"`
	Error   string
}
